package main

import (
	"context"
	"testing"

	"gitflame-codepilot/backend/internal/agent"
	"gitflame-codepilot/backend/internal/domain"
	"gitflame-codepilot/backend/internal/queue"
)

type fakeExecutor struct{ err error }

func (e fakeExecutor) ExecuteTask(context.Context, domain.AgentJob) error { return e.err }
func (e fakeExecutor) RetryTask(domain.AgentJob) error                    { return nil }

type fakeBroker struct {
	published []domain.AgentJob
	acked     []string
	dead      int
}

func (b *fakeBroker) Ping(context.Context) error { return nil }
func (b *fakeBroker) Publish(_ context.Context, j domain.AgentJob) error {
	b.published = append(b.published, j)
	return nil
}
func (b *fakeBroker) EnsureGroup(context.Context) error                    { return nil }
func (b *fakeBroker) Read(context.Context, string) (*queue.Message, error) { return nil, nil }
func (b *fakeBroker) Ack(_ context.Context, id string) error {
	b.acked = append(b.acked, id)
	return nil
}
func (b *fakeBroker) DeadLetter(context.Context, queue.Message, error) error { b.dead++; return nil }

func TestProcessRetriesTemporaryFailure(t *testing.T) {
	broker := &fakeBroker{}
	message := queue.Message{ID: "1-0", Job: domain.AgentJob{TaskID: "task", Attempt: 1}}
	process(context.Background(), fakeExecutor{err: &agent.Error{Status: 503, Code: "model_unavailable", Detail: "loading"}}, broker, message, 3)
	if len(broker.published) != 1 || broker.published[0].Attempt != 2 || len(broker.acked) != 1 || broker.dead != 0 {
		t.Fatalf("unexpected broker state: %+v", broker)
	}
}

func TestProcessDeadLettersPermanentFailure(t *testing.T) {
	broker := &fakeBroker{}
	message := queue.Message{ID: "2-0", Job: domain.AgentJob{TaskID: "task", Attempt: 1}}
	process(context.Background(), fakeExecutor{err: &agent.Error{Status: 502, Code: "invalid_output", Detail: "bad plan"}}, broker, message, 3)
	if broker.dead != 1 || len(broker.acked) != 1 || len(broker.published) != 0 {
		t.Fatalf("unexpected broker state: %+v", broker)
	}
}
