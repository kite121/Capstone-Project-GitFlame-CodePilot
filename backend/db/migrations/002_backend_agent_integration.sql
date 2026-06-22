ALTER TABLE issue_sessions
    ADD COLUMN IF NOT EXISTS request_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE agent_tasks
    ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS error_json JSONB,
    ADD COLUMN IF NOT EXISTS relevant_files JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS usage_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE agent_tasks DROP CONSTRAINT IF EXISTS agent_tasks_attempt_positive;
ALTER TABLE agent_tasks
    ADD CONSTRAINT agent_tasks_attempt_positive CHECK (attempt > 0);
