package agent

import (
	"context"
	"encoding/json"
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
		for _, expected := range []string{`"request_id":"task-1"`, `"configuration_yaml"`, `"repository_files"`, `"previous_plan"`, `"correction_feedback"`} {
			if !strings.Contains(text, expected) {
				t.Fatalf("request body misses %s: %s", expected, text)
			}
		}
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if _, exists := payload["configuration"]; exists {
			t.Fatal("obsolete configuration object was sent")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"request_id":"task-1","status":"completed","plan_markdown":"# Valid plan","relevant_files":[{"path":"main.go","reason":"entrypoint","create":false}],"model":"test-model","usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"tool_calls":1,"reasoning_chars":20,"generation_time_seconds":1.25}}`)), Header: make(http.Header)}, nil
	})
	previous, feedback := "old", "fix tests"
	result, err := client.GeneratePlan(context.Background(), domain.AgentPlanRequest{RequestID: "task-1", Issue: domain.AgentIssue{ID: "1"}, ConfigurationYAML: "version: 1", RepositoryFiles: []domain.RepositoryFile{{Path: "main.go", Content: "package main"}}, PreviousPlan: &previous, CorrectionFeedback: &feedback})
	if err != nil || result.PlanMarkdown != "# Valid plan" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result.Usage.TotalTokens != 15 || result.Usage.ReasoningChars != 20 || len(result.RelevantFiles) != 1 {
		t.Fatalf("Agent Engine metadata was not decoded: %+v", result)
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
