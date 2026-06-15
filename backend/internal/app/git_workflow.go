package app

import (
	"regexp"
	"strings"
)

type GitWorkflowService interface {
	CreatePullRequest(request IssueAnalyzeRequest, cfg AIConfig) (GitWorkflowResponse, error)
}

type MockGitWorkflowService struct{}

func NewMockGitWorkflowService() *MockGitWorkflowService {
	return &MockGitWorkflowService{}
}

func (s *MockGitWorkflowService) CreatePullRequest(request IssueAnalyzeRequest, cfg AIConfig) (GitWorkflowResponse, error) {
	slug := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(request.Issue.Title, "-")
	slug = strings.Trim(strings.ToLower(slug), "-")
	if len(slug) > 48 {
		slug = slug[:48]
	}
	if slug == "" {
		slug = strings.ToLower(request.Issue.ID)
	}

	baseURL := request.Repository.WebURL
	if baseURL == "" {
		baseURL = "https://gitflame.local/" + request.Repository.ID
	}

	reviewer := request.Issue.Author
	if cfg.ReviewerPolicy != "issue_author" {
		reviewer = cfg.ReviewerPolicy
	}

	return GitWorkflowResponse{
		BranchName:     cfg.TargetBranchPrefix + request.Issue.ID + "-" + slug,
		PullRequestURL: strings.TrimRight(baseURL, "/") + "/-/merge_requests/mock-" + request.Issue.ID,
		Reviewer:       reviewer,
		Provider:       "mock",
	}, nil
}
