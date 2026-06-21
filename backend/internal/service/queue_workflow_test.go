package service

import (
	"context"
	"testing"

	"gitflame-codepilot/backend/internal/domain"
	"gitflame-codepilot/backend/internal/queue"
	"gitflame-codepilot/backend/internal/repository"
)

type recordingBroker struct{ jobs []domain.AgentJob }

func (b *recordingBroker) Ping(context.Context) error { return nil }
func (b *recordingBroker) Publish(_ context.Context, job domain.AgentJob) error {
	b.jobs = append(b.jobs, job)
	return nil
}
func (b *recordingBroker) EnsureGroup(context.Context) error                      { return nil }
func (b *recordingBroker) Read(context.Context, string) (*queue.Message, error)   { return nil, nil }
func (b *recordingBroker) Ack(context.Context, string) error                      { return nil }
func (b *recordingBroker) DeadLetter(context.Context, queue.Message, error) error { return nil }

func TestAnalyzeIsIdempotentAndPublishesOneJob(t *testing.T) {
	broker := &recordingBroker{}
	workflow := NewQueuedWorkflow(repository.NewMemoryStore(), broker)
	req := domain.IssueAnalyzeRequest{Repository: domain.RepositoryMetadata{ID: "repo", DefaultBranch: "main"}, Issue: domain.IssuePayload{ID: "42", Title: "Task", Body: "Body", Author: "artur"}, YAMLConfig: "version: 1", RepositoryFiles: []domain.RepositoryFile{{Path: "main.go", Content: "package main"}}}
	firstSession, firstTask, err := workflow.Analyze(req)
	if err != nil {
		t.Fatal(err)
	}
	secondSession, secondTask, err := workflow.Analyze(req)
	if err != nil {
		t.Fatal(err)
	}
	if firstSession.ID != secondSession.ID || firstTask.ID != secondTask.ID {
		t.Fatalf("duplicate resources were created")
	}
	if len(broker.jobs) != 1 {
		t.Fatalf("published %d jobs, want 1", len(broker.jobs))
	}
	job := broker.jobs[0]
	if job.TaskID != firstTask.ID || job.Request.RequestID != firstTask.ID || job.Request.Configuration.MaxFiles != 20 {
		t.Fatalf("unexpected job: %+v", job)
	}
}
