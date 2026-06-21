package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"gitflame-codepilot/backend/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGeneratePlanUsesSprint2Contract(t *testing.T) {
	client := NewClient("http://agent-engine:8001/", time.Second)
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/plans/generate" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		body, _ := io.ReadAll(req.Body)
		text := string(body)
		for _, expected := range []string{`"request_id":"task-1"`, `"configuration"`, `"repository_files"`, `"previous_plan"`, `"correction_feedback"`} {
			if !strings.Contains(text, expected) {
				t.Fatalf("request body misses %s: %s", expected, text)
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"request_id":"task-1","status":"completed","plan_markdown":"# Valid plan","relevant_files":[],"model":"test-model","usage":{"prompt_tokens":10,"completion_tokens":5,"tool_calls":1}}`)), Header: make(http.Header)}, nil
	})
	previous, feedback := "old", "fix tests"
	result, err := client.GeneratePlan(context.Background(), domain.AgentPlanRequest{RequestID: "task-1", Issue: domain.AgentIssue{ID: "1"}, Configuration: domain.AgentConfiguration{MaxFiles: 20}, RepositoryFiles: []domain.RepositoryFile{{Path: "main.go", Content: "package main"}}, PreviousPlan: &previous, CorrectionFeedback: &feedback})
	if err != nil || result.PlanMarkdown != "# Valid plan" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGeneratePlanPreservesSupportedAgentStatus(t *testing.T) {
	client := NewClient("http://agent-engine:8001", time.Second)
	client.httpClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 504, Body: io.NopCloser(strings.NewReader(`{"code":"inference_timeout","detail":"too slow"}`)), Header: make(http.Header)}, nil
	})
	_, err := client.GeneratePlan(context.Background(), domain.AgentPlanRequest{})
	agentErr, ok := err.(*Error)
	if !ok || agentErr.Status != 504 || agentErr.Code != "inference_timeout" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
