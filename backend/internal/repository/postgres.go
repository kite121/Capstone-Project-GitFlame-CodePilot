package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitflame-codepilot/backend/internal/domain"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{pool: pool}
	if err := store.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Close() { s.pool.Close() }
func (s *PostgresStore) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return err
	}
	var ready bool
	if err := s.pool.QueryRow(ctx, `SELECT to_regclass('public.agent_tasks') IS NOT NULL`).Scan(&ready); err != nil {
		return err
	}
	if !ready {
		return errors.New("Sprint 2 database migration is not applied")
	}
	return nil
}

func (s *PostgresStore) CreateSession(req domain.IssueAnalyzeRequest, cfg domain.AIConfig) (*domain.IssueSession, bool, error) {
	if existing, err := s.sessionByRepositoryIssue(req.Repository.ID, req.Issue.ID); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	requestJSON, err := json.Marshal(req)
	if err != nil {
		return nil, false, err
	}
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, false, err
	}
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(context.Background())
	name := req.Repository.Name
	if strings.TrimSpace(name) == "" {
		name = req.Repository.ID
	}
	var repositoryID string
	err = tx.QueryRow(context.Background(), `
		INSERT INTO repositories (external_id, name, owner, default_branch)
		VALUES ($1, $2, 'gitflame', $3)
		ON CONFLICT (external_id) DO UPDATE SET name=EXCLUDED.name, default_branch=EXCLUDED.default_branch
		RETURNING id::text`, req.Repository.ID, name, req.Repository.DefaultBranch).Scan(&repositoryID)
	if err != nil {
		return nil, false, err
	}
	var configID string
	err = tx.QueryRow(context.Background(), `
		INSERT INTO ai_configs (repository_id, raw_yml, parsed_config_json, is_valid)
		VALUES ($1::uuid, $2, $3, true) RETURNING id::text`, repositoryID, cfg.Raw, string(configJSON)).Scan(&configID)
	if err != nil {
		return nil, false, err
	}
	sessionID := NewID()
	command, err := tx.Exec(context.Background(), `
		INSERT INTO issue_sessions (
			id, repository_id, ai_config_id, external_issue_id, issue_title, issue_body,
			issue_author, status, request_json, config_json, revision
		) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10,0)
		ON CONFLICT (repository_id, external_issue_id) WHERE external_issue_id IS NOT NULL DO NOTHING`,
		sessionID, repositoryID, configID, req.Issue.ID, req.Issue.Title, req.Issue.Body,
		req.Issue.Author, domain.SessionGenerating, string(requestJSON), string(configJSON))
	if err != nil {
		return nil, false, err
	}
	if command.RowsAffected() == 0 {
		_ = tx.Rollback(context.Background())
		existing, err := s.sessionByRepositoryIssue(req.Repository.ID, req.Issue.ID)
		return existing, false, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, false, err
	}
	createdSession, err := s.Session(sessionID)
	return createdSession, true, err
}

func (s *PostgresStore) Session(id string) (*domain.IssueSession, error) {
	return scanSession(s.pool.QueryRow(context.Background(), `
		SELECT id::text, status, request_json, config_json, plan_markdown, revision,
		       feedback_history, generated_files, created_at, updated_at
		FROM issue_sessions
		WHERE id::text=$1 OR external_issue_id=$1
		ORDER BY updated_at DESC LIMIT 1`, id))
}

func (s *PostgresStore) sessionByRepositoryIssue(repositoryID, issueID string) (*domain.IssueSession, error) {
	return scanSession(s.pool.QueryRow(context.Background(), `
		SELECT s.id::text, s.status, s.request_json, s.config_json, s.plan_markdown, s.revision,
		       s.feedback_history, s.generated_files, s.created_at, s.updated_at
		FROM issue_sessions s
		JOIN repositories r ON r.id=s.repository_id
		WHERE r.external_id=$1 AND s.external_issue_id=$2
		LIMIT 1`, repositoryID, issueID))
}

func scanSession(row pgx.Row) (*domain.IssueSession, error) {
	var session domain.IssueSession
	var requestJSON, configJSON, feedbackJSON []byte
	var generatedJSON []byte
	err := row.Scan(&session.ID, &session.Status, &requestJSON, &configJSON, &session.PlanMarkdown,
		&session.Revision, &feedbackJSON, &generatedJSON, &session.CreatedAt, &session.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(requestJSON, &session.Request); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configJSON, &session.Config); err != nil {
		return nil, err
	}
	if len(feedbackJSON) > 0 {
		_ = json.Unmarshal(feedbackJSON, &session.FeedbackHistory)
	}
	if len(generatedJSON) > 0 && string(generatedJSON) != "null" {
		_ = json.Unmarshal(generatedJSON, &session.GeneratedFiles)
	}
	return &session, nil
}

func (s *PostgresStore) UpdateSession(session *domain.IssueSession) error {
	feedbackJSON, _ := json.Marshal(session.FeedbackHistory)
	generatedJSON, _ := json.Marshal(session.GeneratedFiles)
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	command, err := tx.Exec(context.Background(), `
		UPDATE issue_sessions SET status=$2, plan_markdown=$3, revision=$4,
		feedback_history=$5, generated_files=$6, updated_at=now() WHERE id=$1::uuid`,
		session.ID, session.Status, session.PlanMarkdown, session.Revision, string(feedbackJSON), string(generatedJSON))
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	if session.Revision > 0 && strings.TrimSpace(session.PlanMarkdown) != "" {
		var feedback any
		if len(session.FeedbackHistory) > 0 {
			feedback = session.FeedbackHistory[len(session.FeedbackHistory)-1]
		}
		_, err = tx.Exec(context.Background(), `
			INSERT INTO plan_revisions (issue_session_id, revision, plan_markdown, correction_feedback)
			VALUES ($1::uuid,$2,$3,$4)
			ON CONFLICT (issue_session_id, revision) DO UPDATE
			SET plan_markdown=EXCLUDED.plan_markdown, correction_feedback=EXCLUDED.correction_feedback`,
			session.ID, session.Revision, session.PlanMarkdown, feedback)
		if err != nil {
			return err
		}
	}
	return tx.Commit(context.Background())
}

func (s *PostgresStore) CreateTask(sessionID, issueID, taskType string) (*domain.AgentTask, error) {
	task := &domain.AgentTask{ID: NewID(), SessionID: sessionID, IssueID: issueID, Type: taskType, Status: domain.TaskQueued, Attempt: 1}
	err := s.pool.QueryRow(context.Background(), `
		INSERT INTO agent_tasks (id,issue_session_id,external_issue_id,task_type,status)
		VALUES ($1::uuid,$2::uuid,$3,$4,$5)
		RETURNING created_at,updated_at`, task.ID, sessionID, issueID, taskType, task.Status).Scan(&task.CreatedAt, &task.UpdatedAt)
	return task, err
}

func (s *PostgresStore) Task(id string) (*domain.AgentTask, error) {
	return scanTask(s.pool.QueryRow(context.Background(), taskSelect+` WHERE id::text=$1`, id))
}

func (s *PostgresStore) LatestTask(sessionID string) (*domain.AgentTask, error) {
	return scanTask(s.pool.QueryRow(context.Background(), taskSelect+` WHERE issue_session_id=$1::uuid ORDER BY created_at DESC LIMIT 1`, sessionID))
}

const taskSelect = `SELECT id::text,issue_session_id::text,external_issue_id,task_type,status,attempt,
	plan_markdown,error_json,relevant_files,model,usage_json,tool_execution_summary,created_at,updated_at FROM agent_tasks`

func scanTask(row pgx.Row) (*domain.AgentTask, error) {
	var task domain.AgentTask
	var errorJSON, relevantJSON, usageJSON []byte
	err := row.Scan(&task.ID, &task.SessionID, &task.IssueID, &task.Type, &task.Status, &task.Attempt,
		&task.PlanMarkdown, &errorJSON, &relevantJSON, &task.Model, &usageJSON,
		&task.ToolExecutionSummary, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(errorJSON) > 0 && string(errorJSON) != "null" {
		_ = json.Unmarshal(errorJSON, &task.Error)
	}
	_ = json.Unmarshal(relevantJSON, &task.RelevantFiles)
	_ = json.Unmarshal(usageJSON, &task.Usage)
	return &task, nil
}

func (s *PostgresStore) UpdateTask(task *domain.AgentTask) error {
	errorJSON, _ := json.Marshal(task.Error)
	relevantJSON, _ := json.Marshal(task.RelevantFiles)
	usageJSON, _ := json.Marshal(task.Usage)
	command, err := s.pool.Exec(context.Background(), `
		UPDATE agent_tasks SET status=$2,attempt=$3,plan_markdown=$4,error_json=$5,relevant_files=$6,
		model=$7,usage_json=$8,tool_execution_summary=$9,updated_at=now() WHERE id=$1::uuid`,
		task.ID, task.Status, task.Attempt, task.PlanMarkdown, string(errorJSON), string(relevantJSON),
		task.Model, string(usageJSON), task.ToolExecutionSummary)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) SaveRecommendations(id, summary string, cards []domain.RecommendationCard) (*domain.RecommendationReport, error) {
	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	var repositoryID string
	err = tx.QueryRow(context.Background(), `
		INSERT INTO repositories (external_id,name,owner,default_branch)
		VALUES ($1,$1,'gitflame','main')
		ON CONFLICT (external_id) DO UPDATE SET external_id=EXCLUDED.external_id
		RETURNING id::text`, id).Scan(&repositoryID)
	if err != nil {
		return nil, err
	}
	var configID string
	err = tx.QueryRow(context.Background(), `
		INSERT INTO ai_configs (repository_id,raw_yml,parsed_config_json,is_valid)
		VALUES ($1::uuid,'version: 1','{}'::jsonb,true) RETURNING id::text`, repositoryID).Scan(&configID)
	if err != nil {
		return nil, err
	}
	var runID string
	err = tx.QueryRow(context.Background(), `
		INSERT INTO recommendation_runs (repository_id,ai_config_id,summary,status)
		VALUES ($1::uuid,$2::uuid,$3,'completed') RETURNING id::text`, repositoryID, configID, summary).Scan(&runID)
	if err != nil {
		return nil, err
	}
	for index := range cards {
		if cards[index].ID == "" {
			cards[index].ID = NewID()
		}
		if cards[index].State == "" {
			cards[index].State = "open"
		}
		_, err = tx.Exec(context.Background(), `
			INSERT INTO recommendations (id,recommendation_run_id,file_path,line_number,category,severity,problem,suggestion,current_status,confidence)
			VALUES ($1::uuid,$2::uuid,$3,$4,'general',$5,$6,$7,$8,$9)`,
			cards[index].ID, runID, cards[index].File, cards[index].Line, cards[index].Severity,
			cards[index].Problem, cards[index].Suggestion, cards[index].State, cards[index].Confidence)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return &domain.RecommendationReport{RepositoryID: id, Summary: summary, Status: "ready", Recommendations: cards}, nil
}
func (s *PostgresStore) Recommendations(id string) (*domain.RecommendationReport, error) {
	var runID, summary, status string
	err := s.pool.QueryRow(context.Background(), `
		SELECT rr.id::text,rr.summary,rr.status FROM recommendation_runs rr
		JOIN repositories r ON r.id=rr.repository_id
		WHERE r.external_id=$1 AND rr.retention_until>now()
		ORDER BY rr.created_at DESC LIMIT 1`, id).Scan(&runID, &summary, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(context.Background(), `
		SELECT id::text,severity,file_path,line_number,problem,suggestion,confidence,current_status
		FROM recommendations WHERE recommendation_run_id=$1::uuid ORDER BY created_at`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []domain.RecommendationCard
	for rows.Next() {
		var card domain.RecommendationCard
		var line pgtype.Int4
		var confidence pgtype.Float8
		if err := rows.Scan(&card.ID, &card.Severity, &card.File, &line, &card.Problem, &card.Suggestion, &confidence, &card.State); err != nil {
			return nil, err
		}
		if line.Valid {
			value := int(line.Int32)
			card.Line = &value
		}
		if confidence.Valid {
			value := confidence.Float64
			card.Confidence = &value
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if status == "completed" {
		status = "ready"
	}
	return &domain.RecommendationReport{RepositoryID: id, Summary: summary, Status: status, Recommendations: cards}, nil
}
func (s *PostgresStore) CloseRecommendation(id string) (domain.RecommendationCard, error) {
	var card domain.RecommendationCard
	var line pgtype.Int4
	var confidence pgtype.Float8
	err := s.pool.QueryRow(context.Background(), `
		UPDATE recommendations SET current_status='closed' WHERE id::text=$1
		RETURNING id::text,severity,file_path,line_number,problem,suggestion,confidence,current_status`, id).
		Scan(&card.ID, &card.Severity, &card.File, &line, &card.Problem, &card.Suggestion, &confidence, &card.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return card, ErrNotFound
	}
	if line.Valid {
		value := int(line.Int32)
		card.Line = &value
	}
	if confidence.Valid {
		value := confidence.Float64
		card.Confidence = &value
	}
	return card, err
}
func (s *PostgresStore) DeleteRecommendation(id string) error {
	command, err := s.pool.Exec(context.Background(), `DELETE FROM recommendations WHERE id::text=$1`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
