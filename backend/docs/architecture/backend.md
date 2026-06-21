# Go Backend Architecture - Sprint 2

## Package boundaries

Dependencies point inward toward `internal/domain`:

```text
cmd/server
    |
    v
internal/httpapi --> internal/service --> internal/agent
       |                  |
       +-----------------> internal/repository
                              |
                              v
                        internal/domain
```

- `httpapi` owns routes, JSON encoding, status codes, Swagger, and transport DTOs.
- `service` owns issue validation, workflow transitions, and asynchronous orchestration.
- `agent` owns only the HTTP contract with the SERGE-based Agent Engine.
- `repository` hides storage behind an interface. `MemoryStore` supports isolated tests and `PostgresStore` persists the Sprint 2 workflow.
- `queue` owns the Redis Streams transport contract.
- `domain` contains shared workflow entities and contracts without HTTP or database dependencies.

## Issue-to-plan flow

1. GitFlame calls `POST /integrations/gitflame/issues/analyze`.
2. The backend validates issue data, YAML configuration, and repository context.
3. The backend creates a session and an asynchronous task, then returns `202 Accepted`.
4. In queue mode, backend publishes the task to Redis Streams and `agent-worker` calls Agent Engine `POST /v1/plans/generate`.
5. The frontend polls `GET /ai/tasks/{taskId}` until `completed` or `failed`.
6. A correction creates another task and sends `previous_plan` plus `correction_feedback`.
7. Approval changes workflow state and returns a contract for future code generation. It does not create a branch or pull request in Sprint 2.

The Agent Engine request follows the agreed Sprint 2 contract: `request_id`, issue data, repository revision, parsed `configuration`, `repository_files`, and nullable correction fields. The response metadata includes relevant files, model ID, token usage, and tool-call count.

The Redis worker uses a consumer group, concurrency `1`, bounded retries for temporary Agent Engine errors, and a dead-letter stream for permanent failures. PostgreSQL is authoritative; Redis is only the transport. Local mode remains available for tests and development without Redis.
