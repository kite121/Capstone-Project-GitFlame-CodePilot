package queue

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gitflame-codepilot/backend/internal/domain"
)

type RedisBroker struct {
	address, username, password, stream, group, deadLetter string
	database, maxLength                                    int
	dialer                                                 net.Dialer
}

func NewRedisBroker(rawURL, stream, group string, maxLength int) (*RedisBroker, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "redis" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid REDIS_URL")
	}
	database := 0
	if path := strings.TrimPrefix(parsed.Path, "/"); path != "" {
		database, err = strconv.Atoi(path)
		if err != nil || database < 0 {
			return nil, fmt.Errorf("invalid Redis database in REDIS_URL")
		}
	}
	password, _ := parsed.User.Password()
	username := parsed.User.Username()
	address := parsed.Host
	if parsed.Port() == "" {
		address = net.JoinHostPort(parsed.Hostname(), "6379")
	}
	if stream == "" {
		stream = "gitflame:agent:tasks"
	}
	if group == "" {
		group = "gitflame-agent-workers"
	}
	if maxLength < 1 {
		maxLength = 1000
	}
	return &RedisBroker{address: address, username: username, password: password, database: database, stream: stream, group: group, deadLetter: stream + ":dead-letter", maxLength: maxLength}, nil
}

func (r *RedisBroker) Ping(ctx context.Context) error {
	value, err := r.command(ctx, "PING")
	if err != nil {
		return err
	}
	if value != "PONG" {
		return fmt.Errorf("unexpected Redis PING response")
	}
	return nil
}

func (r *RedisBroker) Publish(ctx context.Context, job domain.AgentJob) error {
	length, err := r.command(ctx, "XLEN", r.stream)
	if err != nil {
		return err
	}
	if count, ok := length.(int64); ok && count >= int64(r.maxLength) {
		return ErrQueueFull
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = r.command(ctx, "XADD", r.stream, "*", "job", string(payload))
	return err
}

func (r *RedisBroker) EnsureGroup(ctx context.Context) error {
	_, err := r.command(ctx, "XGROUP", "CREATE", r.stream, r.group, "0", "MKSTREAM")
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (r *RedisBroker) Read(ctx context.Context, consumer string) (*Message, error) {
	claimed, err := r.command(ctx, "XAUTOCLAIM", r.stream, r.group, consumer, "60000", "0-0", "COUNT", "1")
	if err != nil {
		return nil, err
	}
	if response, ok := claimed.([]any); ok && len(response) >= 2 {
		if messages, ok := response[1].([]any); ok && len(messages) > 0 {
			return decodeStreamMessage(messages[0])
		}
	}
	value, err := r.command(ctx, "XREADGROUP", "GROUP", r.group, consumer, "COUNT", "1", "BLOCK", "1000", "STREAMS", r.stream, ">")
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	root, ok := value.([]any)
	if !ok || len(root) == 0 {
		return nil, fmt.Errorf("invalid Redis stream response")
	}
	streamEntry, ok := root[0].([]any)
	if !ok || len(streamEntry) != 2 {
		return nil, fmt.Errorf("invalid Redis stream entry")
	}
	messages, ok := streamEntry[1].([]any)
	if !ok || len(messages) == 0 {
		return nil, nil
	}
	return decodeStreamMessage(messages[0])
}

func decodeStreamMessage(value any) (*Message, error) {
	message, ok := value.([]any)
	if !ok || len(message) != 2 {
		return nil, fmt.Errorf("invalid Redis message")
	}
	id, ok := message[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid Redis message id")
	}
	fields, ok := message[1].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid Redis message fields")
	}
	var payload string
	for index := 0; index+1 < len(fields); index += 2 {
		if fields[index] == "job" {
			payload, _ = fields[index+1].(string)
		}
	}
	if payload == "" {
		return &Message{ID: id}, fmt.Errorf("Redis message does not contain job payload")
	}
	var job domain.AgentJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return &Message{ID: id}, fmt.Errorf("decode Redis job: %w", err)
	}
	return &Message{ID: id, Job: job}, nil
}

func (r *RedisBroker) Ack(ctx context.Context, id string) error {
	_, err := r.command(ctx, "XACK", r.stream, r.group, id)
	if err != nil {
		return err
	}
	_, err = r.command(ctx, "XDEL", r.stream, id)
	return err
}

func (r *RedisBroker) DeadLetter(ctx context.Context, message Message, cause error) error {
	payload, err := json.Marshal(message.Job)
	if err != nil {
		return err
	}
	_, err = r.command(ctx, "XADD", r.deadLetter, "*", "original_id", message.ID, "error", cause.Error(), "job", string(payload))
	return err
}

func (r *RedisBroker) command(ctx context.Context, args ...string) (any, error) {
	connection, err := r.dialer.DialContext(ctx, "tcp", r.address)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	reader := bufio.NewReader(connection)
	if r.password != "" {
		auth := []string{"AUTH", r.password}
		if r.username != "" {
			auth = []string{"AUTH", r.username, r.password}
		}
		if err := writeCommand(connection, auth...); err != nil {
			return nil, err
		}
		if _, err := readRESP(reader); err != nil {
			return nil, err
		}
	}
	if r.database != 0 {
		if err := writeCommand(connection, "SELECT", strconv.Itoa(r.database)); err != nil {
			return nil, err
		}
		if _, err := readRESP(reader); err != nil {
			return nil, err
		}
	}
	if err := writeCommand(connection, args...); err != nil {
		return nil, err
	}
	return readRESP(reader)
}

func writeCommand(writer io.Writer, args ...string) error {
	if _, err := fmt.Fprintf(writer, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, argument := range args {
		if _, err := fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(argument), argument); err != nil {
			return err
		}
	}
	return nil
}

func readRESP(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	line := func() (string, error) {
		value, err := reader.ReadString('\n')
		return strings.TrimSuffix(strings.TrimSuffix(value, "\n"), "\r"), err
	}
	switch prefix {
	case '+':
		return line()
	case '-':
		value, err := line()
		if err != nil {
			return nil, err
		}
		return nil, errors.New(value)
	case ':':
		value, err := line()
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(value, 10, 64)
	case '$':
		value, err := line()
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		buffer := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return nil, err
		}
		return string(buffer[:length]), nil
	case '*':
		value, err := line()
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		result := make([]any, length)
		for index := range result {
			result[index], err = readRESP(reader)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported Redis response prefix %q", prefix)
	}
}
