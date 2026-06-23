package repository

import (
	"context"
	"os"
	"testing"

	"gitflame-codepilot/backend/internal/domain"
)

func TestPostgresIssueTaskPersistence(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration test")
	}
	store, err := NewPostgresStore(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	req := domain.IssueAnalyzeRequest{Repository: domain.RepositoryMetadata{ID: "integration-repo", DefaultBranch: "main"}, Issue: domain.IssuePayload{ID: NewID(), Title: "Persistence test", Body: "Verify task persistence", Author: "test"}, YAMLConfig: "version: 1", RepositoryFiles: []domain.RepositoryFile{{Path: "main.go", Content: "package main"}}}
	session, created, err := store.CreateSession(req, domain.AIConfig{Raw: "version: 1", Version: "1", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected new session")
	}
	task, err := store.CreateTask(session.ID, req.Issue.ID, "initial_plan")
	if err != nil {
		t.Fatal(err)
	}
	task.Status = domain.TaskProcessing
	if err := store.UpdateTask(task); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Task(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != domain.TaskProcessing {
		t.Fatalf("status=%s", loaded.Status)
	}
}
