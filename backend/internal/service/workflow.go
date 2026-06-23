package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"gitflame-codepilot/backend/internal/agent"
	"gitflame-codepilot/backend/internal/domain"
	"gitflame-codepilot/backend/internal/queue"
	"gitflame-codepilot/backend/internal/repository"
)

type Workflow struct {
	store     repository.Store
	generator agent.PlanGenerator
	broker    queue.Broker
}

func NewWorkflow(store repository.Store, generator agent.PlanGenerator) *Workflow {
	return &Workflow{store: store, generator: generator}
}

func NewQueuedWorkflow(store repository.Store, broker queue.Broker) *Workflow {
	return &Workflow{store: store, broker: broker}
}

func (w *Workflow) Analyze(req domain.IssueAnalyzeRequest) (*domain.IssueSession, *domain.AgentTask, error) {
	if err := ValidateAnalyze(req); err != nil {
		return nil, nil, err
	}
	cfg, err := ParseAIConfig(req.YAMLConfig)
	if err != nil {
		return nil, nil, err
	}
	session, created, err := w.store.CreateSession(req, cfg)
	if err != nil {
		return nil, nil, err
	}
	if !created {
		task, err := w.store.LatestTask(session.ID)
		if err == nil {
			return session, task, nil
		}
		if !errors.Is(err, repository.ErrNotFound) || session.Status != domain.SessionGenerating {
			return nil, nil, err
		}
	}
	task, err := w.store.CreateTask(session.ID, req.Issue.ID, "initial_plan")
	if err != nil {
		return nil, nil, err
	}
	job := domain.AgentJob{TaskID: task.ID, SessionID: session.ID, Type: task.Type, Attempt: 1, Request: agentRequest(task.ID, session, nil, nil)}
	if err := w.dispatch(job); err != nil {
		return nil, nil, err
	}
	return session, task, nil
}

func (w *Workflow) Correct(issueID, feedback string) (*domain.AgentTask, error) {
	if strings.TrimSpace(feedback) == "" {
		return nil, errors.New("feedback is required")
	}
	session, err := w.store.Session(issueID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.SessionPlanGenerated {
		return nil, fmt.Errorf("plan cannot be corrected while session status is %s", session.Status)
	}
	session.Status = domain.SessionCorrectionRequested
	session.FeedbackHistory = append(session.FeedbackHistory, feedback)
	if err := w.store.UpdateSession(session); err != nil {
		return nil, err
	}
	task, err := w.store.CreateTask(session.ID, session.Request.Issue.ID, "plan_revision")
	if err != nil {
		return nil, err
	}
	previousPlan := session.PlanMarkdown
	job := domain.AgentJob{TaskID: task.ID, SessionID: session.ID, Type: task.Type, Attempt: 1, Request: agentRequest(task.ID, session, &previousPlan, &feedback)}
	if err := w.dispatch(job); err != nil {
		return nil, err
	}
	return task, nil
}

func (w *Workflow) Retry(taskID string) (*domain.AgentTask, error) {
	failed, err := w.store.Task(taskID)
	if err != nil {
		return nil, err
	}
	if failed.Status != domain.TaskFailed || failed.Error == nil || !recoverableTaskError(failed.Error) {
		return nil, errors.New("only failed tasks with a temporary Agent Engine error can be retried")
	}
	session, err := w.store.Session(failed.SessionID)
	if err != nil {
		return nil, err
	}
	task, err := w.store.CreateTask(session.ID, session.Request.Issue.ID, failed.Type)
	if err != nil {
		return nil, err
	}
	task.Attempt = failed.Attempt + 1
	if err := w.store.UpdateTask(task); err != nil {
		return nil, err
	}
	session.Status = domain.SessionGenerating
	if err := w.store.UpdateSession(session); err != nil {
		return nil, err
	}
	var previousPlan, feedback *string
	if failed.Type == "plan_revision" {
		previous := session.PlanMarkdown
		previousPlan = &previous
		if len(session.FeedbackHistory) > 0 {
			value := session.FeedbackHistory[len(session.FeedbackHistory)-1]
			feedback = &value
		}
	}
	job := domain.AgentJob{TaskID: task.ID, SessionID: session.ID, Type: task.Type, Attempt: task.Attempt, Request: agentRequest(task.ID, session, previousPlan, feedback)}
	if err := w.dispatch(job); err != nil {
		return nil, err
	}
	return task, nil
}

func (w *Workflow) dispatch(job domain.AgentJob) error {
	if w.broker != nil {
		if err := w.broker.Publish(context.Background(), job); err != nil {
			_ = w.failTask(job.TaskID, err)
			return fmt.Errorf("%w: %v", ErrDispatch, err)
		}
		return nil
	}
	if w.generator == nil {
		return fmt.Errorf("%w: task dispatcher is not configured", ErrDispatch)
	}
	go func() { _ = w.ExecuteTask(context.Background(), job) }()
	return nil
}

func (w *Workflow) ExecuteTask(ctx context.Context, job domain.AgentJob) error {
	if w.generator == nil {
		return errors.New("Agent Engine client is not configured")
	}
	task, err := w.store.Task(job.TaskID)
	if err != nil {
		return err
	}
	task.Status = domain.TaskProcessing
	task.Attempt = job.Attempt
	task.Error = nil
	if err := w.store.UpdateTask(task); err != nil {
		return err
	}
	session, err := w.store.Session(task.SessionID)
	if err != nil {
		return err
	}
	session.Status = domain.SessionProcessing
	if err := w.store.UpdateSession(session); err != nil {
		return err
	}
	result, err := w.generator.GeneratePlan(ctx, job.Request)
	if err != nil {
		_ = w.failTask(job.TaskID, err)
		return err
	}
	validationFiles := append([]domain.RepositoryFile(nil), job.Request.RepositoryFiles...)
	for _, relevant := range result.RelevantFiles {
		validationFiles = append(validationFiles, domain.RepositoryFile{Path: relevant.Path})
	}
	if err := ValidatePlan(result.PlanMarkdown, validationFiles); err != nil {
		invalid := &agent.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_output", Detail: err.Error()}
		_ = w.failTask(job.TaskID, invalid)
		return invalid
	}
	task.Status = domain.TaskCompleted
	task.PlanMarkdown = result.PlanMarkdown
	task.RelevantFiles = result.RelevantFiles
	task.Model = result.Model
	task.Usage = result.Usage
	task.ToolExecutionSummary = fmt.Sprintf(
		"model=%s; tool_calls=%d; prompt_tokens=%d; completion_tokens=%d; total_tokens=%d; reasoning_chars=%d; generation_seconds=%.3f",
		result.Model, result.Usage.ToolCalls, result.Usage.PromptTokens, result.Usage.CompletionTokens,
		result.Usage.TotalTokens, result.Usage.ReasoningChars, result.Usage.GenerationTimeSeconds,
	)
	if err := w.store.UpdateTask(task); err != nil {
		return err
	}
	session, err = w.store.Session(task.SessionID)
	if err != nil {
		return err
	}
	session.PlanMarkdown = result.PlanMarkdown
	session.Status = domain.SessionPlanGenerated
	session.Revision++
	return w.store.UpdateSession(session)
}

func (w *Workflow) RetryTask(job domain.AgentJob) error {
	task, err := w.store.Task(job.TaskID)
	if err != nil {
		return err
	}
	task.Status = domain.TaskQueued
	task.Attempt = job.Attempt
	if err := w.store.UpdateTask(task); err != nil {
		return err
	}
	session, err := w.store.Session(task.SessionID)
	if err != nil {
		return err
	}
	session.Status = domain.SessionGenerating
	return w.store.UpdateSession(session)
}

func (w *Workflow) failTask(taskID string, cause error) error {
	task, err := w.store.Task(taskID)
	if err != nil {
		return err
	}
	task.Status = domain.TaskFailed
	task.Error = toTaskError(cause)
	if err := w.store.UpdateTask(task); err != nil {
		return err
	}
	session, err := w.store.Session(task.SessionID)
	if err == nil {
		session.Status = domain.SessionFailed
		_ = w.store.UpdateSession(session)
	}
	return nil
}

func (w *Workflow) Task(id string) (*domain.AgentTask, error)       { return w.store.Task(id) }
func (w *Workflow) Session(id string) (*domain.IssueSession, error) { return w.store.Session(id) }

func (w *Workflow) Approve(issueID string) (*domain.IssueSession, error) {
	session, err := w.store.Session(issueID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.SessionPlanGenerated {
		return nil, fmt.Errorf("plan cannot be approved while session status is %s", session.Status)
	}
	slug := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(strings.ToLower(session.Request.Issue.Title), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	branch := session.Config.TargetBranchPrefix + session.Request.Issue.ID + "-" + slug
	session.GeneratedFiles = &domain.GeneratedFilesContract{BranchName: branch, Files: []domain.GeneratedFileOperation{}, CommitMessage: "Implement " + session.Request.Issue.Title, PRTitle: session.Request.Issue.Title, Reviewer: session.Request.Issue.Author}
	session.Status = domain.SessionApproved
	if err := w.store.UpdateSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (w *Workflow) Reject(issueID string) (*domain.IssueSession, error) {
	session, err := w.store.Session(issueID)
	if err != nil {
		return nil, err
	}
	if session.Status == domain.SessionApproved {
		return nil, errors.New("approved plan cannot be rejected")
	}
	session.Status = domain.SessionRejected
	if err := w.store.UpdateSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func agentRequest(taskID string, session *domain.IssueSession, previousPlan, feedback *string) domain.AgentPlanRequest {
	files := append([]domain.RepositoryFile(nil), session.Request.RepositoryFiles...)
	if len(files) == 0 {
		for _, path := range session.Request.RepositoryContext {
			files = append(files, domain.RepositoryFile{Path: path})
		}
	}
	return domain.AgentPlanRequest{
		RequestID:         taskID,
		Issue:             domain.AgentIssue{ID: session.Request.Issue.ID, Title: session.Request.Issue.Title, Body: session.Request.Issue.Body},
		Repository:        domain.AgentRepository{ID: session.Request.Repository.ID, DefaultBranch: session.Request.Repository.DefaultBranch, CommitSHA: session.Request.Repository.CommitSHA},
		ConfigurationYAML: session.Config.Raw,
		RepositoryFiles:   files, PreviousPlan: previousPlan, CorrectionFeedback: feedback,
	}
}

func ValidateAnalyze(v domain.IssueAnalyzeRequest) error {
	switch {
	case strings.TrimSpace(v.Repository.ID) == "":
		return errors.New("repository.id is required")
	case strings.TrimSpace(v.Repository.DefaultBranch) == "":
		return errors.New("repository.default_branch is required")
	case strings.TrimSpace(v.Issue.ID) == "":
		return errors.New("issue.id is required")
	case strings.TrimSpace(v.Issue.Title) == "":
		return errors.New("issue.title is required")
	case strings.TrimSpace(v.Issue.Body) == "":
		return errors.New("issue.body is required")
	case strings.TrimSpace(v.Issue.Author) == "":
		return errors.New("issue.author is required")
	case len(v.RepositoryFiles) == 0 && len(v.RepositoryContext) == 0:
		return errors.New("repository_files must contain at least one file")
	default:
		files := v.RepositoryFiles
		if len(files) == 0 {
			for _, legacyPath := range v.RepositoryContext {
				files = append(files, domain.RepositoryFile{Path: legacyPath})
			}
		}
		return validateRepositoryFiles(files)
	}
}

func validateRepositoryFiles(files []domain.RepositoryFile) error {
	if len(files) > 2_000 {
		return errors.New("repository_files cannot contain more than 2000 files")
	}
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		normalized := normalizePlanPath(file.Path)
		if strings.TrimSpace(file.Path) == "" || !safePlanPath(normalized) {
			return fmt.Errorf("repository_files contains unsafe path %q", file.Path)
		}
		if len(file.Content) > 500_000 {
			return fmt.Errorf("repository file %q exceeds 500000 characters", normalized)
		}
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("repository_files contains duplicate path %q", normalized)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func toTaskError(err error) *domain.TaskError {
	var engineError *agent.Error
	if errors.As(err, &engineError) {
		return &domain.TaskError{HTTPStatus: engineError.Status, Code: engineError.Code, Detail: engineError.Detail}
	}
	return &domain.TaskError{HTTPStatus: http.StatusBadGateway, Code: "agent_engine_error", Detail: err.Error()}
}

func recoverableTaskError(taskError *domain.TaskError) bool {
	if taskError.HTTPStatus == http.StatusServiceUnavailable || taskError.HTTPStatus == http.StatusGatewayTimeout {
		return true
	}
	return taskError.HTTPStatus == http.StatusBadGateway &&
		(taskError.Code == "agent_engine_unreachable" || taskError.Code == "agent_engine_error")
}

var ErrDispatch = errors.New("task dispatch failed")
