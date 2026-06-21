package queue

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"gitflame-codepilot/backend/internal/domain"
)

func TestRedisStreamLifecycle(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("set TEST_REDIS_URL to run Redis integration test")
	}
	name := "gitflame:test:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	broker, err := NewRedisBroker(redisURL, name, "test-workers", 10)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := broker.EnsureGroup(ctx); err != nil {
		t.Fatal(err)
	}
	job := domain.AgentJob{TaskID: "task-1", SessionID: "session-1", Type: "generate", Attempt: 1, Request: domain.AgentPlanRequest{RequestID: "task-1"}}
	if err := broker.Publish(ctx, job); err != nil {
		t.Fatal(err)
	}
	message, err := broker.Read(ctx, "test-consumer")
	if err != nil {
		t.Fatal(err)
	}
	if message == nil || message.Job.TaskID != job.TaskID {
		t.Fatalf("message=%+v", message)
	}
	if err := broker.Ack(ctx, message.ID); err != nil {
		t.Fatal(err)
	}
}
