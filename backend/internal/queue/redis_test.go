package queue

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

func TestRESPRoundTripHelpers(t *testing.T) {
	var command bytes.Buffer
	if err := writeCommand(&command, "XADD", "tasks", "*", "job", `{}`); err != nil {
		t.Fatal(err)
	}
	expected := "*5\r\n$4\r\nXADD\r\n$5\r\ntasks\r\n$1\r\n*\r\n$3\r\njob\r\n$2\r\n{}\r\n"
	if command.String() != expected {
		t.Fatalf("command mismatch:\n%q", command.String())
	}

	value, err := readRESP(bufio.NewReader(bytes.NewBufferString("*2\r\n$6\r\nstream\r\n*2\r\n:1\r\n$3\r\njob\r\n")))
	if err != nil {
		t.Fatal(err)
	}
	expectedValue := []any{"stream", []any{int64(1), "job"}}
	if !reflect.DeepEqual(value, expectedValue) {
		t.Fatalf("response=%#v", value)
	}
}

func TestRedisURLValidation(t *testing.T) {
	if _, err := NewRedisBroker("http://localhost:6379", "", "", 0); err == nil {
		t.Fatal("expected invalid scheme error")
	}
	broker, err := NewRedisBroker("redis://:secret@localhost:6379/2", "tasks", "workers", 10)
	if err != nil {
		t.Fatal(err)
	}
	if broker.database != 2 || broker.password != "secret" || broker.deadLetter != "tasks:dead-letter" {
		t.Fatalf("unexpected broker: %+v", broker)
	}
}
