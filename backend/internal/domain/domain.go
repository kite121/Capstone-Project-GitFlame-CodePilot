package domain

import "time"

const (
	TaskQueued     = "queued"
	TaskProcessing = "processing"
	TaskCompleted  = "completed"
	TaskFailed     = "failed"

	SessionGenerating          = "generating"
	SessionPlanGenerated       = "plan_generated"
	SessionApproved            = "approved"
	SessionCorrectionRequested = "correction_requested"
	SessionRejected            = "rejected"
)

type RepositoryMetadata struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	DefaultBranch string `json:"default_branch"`
	CommitSHA     string `json:"commit_sha,omitempty"`
	WebURL        string `json:"web_url,omitempty"`
}

type IssuePayload struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	Author string `json:"author"`
}

type IssueAnalyzeRequest struct {
	Repository      RepositoryMetadata `json:"repository"`
	Issue           IssuePayload       `json:"issue"`
	YAMLConfig      string             `json:"yaml_config"`
	RepositoryFiles []RepositoryFile   `json:"repository_files"`
	// RepositoryContext is retained temporarily for Sprint 1 clients.
	RepositoryContext []string       `json:"repository_context,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type RepositoryFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type AIConfig struct {
	Raw, Version, DefaultBranch, TargetBranchPrefix string
	AnalysisEnabled, RequireApproval                bool
	IncludePatterns, ExcludePatterns                []string
	MaxFiles, MaxSnippetsPerFile                    int
	ReviewerPolicy                                  string
	ApproveCommand, CorrectCommand, RejectCommand   string
}

type IssueSession struct {
	ID, Status, PlanMarkdown string
	Request                  IssueAnalyzeRequest
	Config                   AIConfig
	Revision                 int
	FeedbackHistory          []string
	GeneratedFiles           *GeneratedFilesContract
	CreatedAt, UpdatedAt     time.Time
}

type AgentTask struct {
	ID, SessionID, IssueID, Type, Status string
	Attempt                              int
	PlanMarkdown                         string
	Error                                *TaskError
	ToolExecutionSummary                 string
	RelevantFiles                        []RelevantFile
	Model                                string
	Usage                                AgentUsage
	CreatedAt, UpdatedAt                 time.Time
}

type AgentJob struct {
	TaskID    string           `json:"task_id"`
	SessionID string           `json:"session_id"`
	Type      string           `json:"type"`
	Attempt   int              `json:"attempt"`
	Request   AgentPlanRequest `json:"request"`
}

type TaskError struct {
	HTTPStatus int    `json:"http_status"`
	Code       string `json:"code"`
	Detail     string `json:"detail"`
}

type AgentPlanRequest struct {
	RequestID          string             `json:"request_id"`
	Issue              AgentIssue         `json:"issue"`
	Repository         AgentRepository    `json:"repository"`
	Configuration      AgentConfiguration `json:"configuration"`
	RepositoryFiles    []RepositoryFile   `json:"repository_files"`
	PreviousPlan       *string            `json:"previous_plan"`
	CorrectionFeedback *string            `json:"correction_feedback"`
}

type AgentPlanResponse struct {
	RequestID     string         `json:"request_id"`
	Status        string         `json:"status"`
	PlanMarkdown  string         `json:"plan_markdown"`
	RelevantFiles []RelevantFile `json:"relevant_files"`
	Model         string         `json:"model"`
	Usage         AgentUsage     `json:"usage"`
}

type AgentIssue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type AgentRepository struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
	CommitSHA     string `json:"commit_sha"`
}

type AgentConfiguration struct {
	Include            []string `json:"include"`
	Exclude            []string `json:"exclude"`
	MaxFiles           int      `json:"max_files"`
	MaxSnippetsPerFile int      `json:"max_snippets_per_file"`
}

type RelevantFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type AgentUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	ToolCalls        int `json:"tool_calls"`
}

type GeneratedFilesContract struct {
	BranchName    string                   `json:"branch_name"`
	Files         []GeneratedFileOperation `json:"files"`
	CommitMessage string                   `json:"commit_message"`
	PRTitle       string                   `json:"pr_title"`
	Reviewer      string                   `json:"reviewer"`
}

type GeneratedFileOperation struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	Content   string `json:"content,omitempty"`
}

type RecommendationCard struct {
	ID         string   `json:"id"`
	Severity   string   `json:"severity"`
	File       string   `json:"file"`
	Line       *int     `json:"line,omitempty"`
	Problem    string   `json:"problem"`
	Suggestion string   `json:"suggestion"`
	Confidence *float64 `json:"confidence,omitempty"`
	State      string   `json:"state"`
}

type RecommendationReport struct {
	RepositoryID, Summary, Status string
	Recommendations               []RecommendationCard
}
