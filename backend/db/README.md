# Backend Database Schema

This folder contains the PostgreSQL schema and migrations for the GitFlame CodePilot backend storage layer.

`schema.sql` creates the Sprint 1 tables:

- `repositories`
- `ai_configs`
- `issue_sessions`
- `generated_plans`
- `user_responses`
- `recommendation_runs`
- `recommendations`
- `recommendation_statuses`
- `plan_revisions`
- `agent_tasks`

The schema uses PostgreSQL-friendly types:

- `UUID` for identifiers
- `JSONB` for parsed `.yml` configuration
- `TIMESTAMPTZ` for timestamps
- `TEXT CHECK (...)` constraints for MVP status values

After the PostgreSQL container is running, the schema can be applied with:

```bash
psql postgresql://gitflame:gitflame@localhost:5432/gitflame_codepilot -f backend/db/schema.sql
```

For a database created from the Sprint 1 schema, apply the Sprint 2 migration:

```bash
psql "$DATABASE_URL" -f backend/db/migrations/002_sprint2_agent_tasks.sql
```

The PostgreSQL integration test is opt-in:

```bash
TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/repository -run TestPostgresIssueTaskPersistence
```
