# Backend Database

PostgreSQL is the authoritative Sprint 2 workflow store.

## Files

- `migrations/initial_schema.sql` - complete schema from the latest database branch, extended with the bounded Agent Engine metadata required by the worker.
- `migrations/002_backend_agent_integration.sql` - adds only backend/worker integration columns to an existing Sprint 2 database.
- `schema.sql` - compatibility copy of the complete schema for older Compose configurations.
- `verification.sql` - sample persistence and verification queries supplied by the database workstream.

The schema stores repositories, validated configuration, issue sessions, current generated plans,
plan revisions, correction feedback, agent task states, user responses, recommendations, and
recommendation retention.

## Fresh database

The latest root Compose mounts `migrations/initial_schema.sql` into PostgreSQL initialization.
For a manual setup:

```bash
psql "$DATABASE_URL" -f backend/db/migrations/initial_schema.sql
```

## Existing Sprint 2 database

If Amir's `initial_schema.sql` was already applied before the backend integration columns were
added, run:

```bash
psql "$DATABASE_URL" -f backend/db/migrations/002_backend_agent_integration.sql
```

The old Sprint 1 schema is not structurally equivalent. Use the complete initial schema for a new
development database instead of applying only migration `002` to Sprint 1 tables.

## Integration test

```bash
TEST_DATABASE_URL="$DATABASE_URL" \
go test ./internal/repository -run TestPostgresIssueTaskPersistence -count=1
```
