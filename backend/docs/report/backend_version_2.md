# Backend Version 2 - Weekly Report Material

## New and changed endpoints

| Endpoint | Change in Version 2 |
| --- | --- |
| `POST /integrations/gitflame/issues/analyze` | Validates issue data, YAML configuration, and repository context; returns `202 Accepted` with an asynchronous task instead of waiting for inference. |
| `GET /ai/tasks/{taskId}` | New endpoint for `queued`, `processing`, `completed`, and `failed` states. Completed tasks include `plan_markdown`; failed tasks include a normalized Agent Engine error. |
| `POST /ai/tasks/{taskId}/retry` | Creates a replacement task only for failures classified as temporary (`502`, `503`, or `504`). |
| `GET /ai/issues/{id}/plan` | Returns the latest completed plan revision; returns `409` while a plan is not ready. |
| `POST /ai/issues/{id}/correct` | Creates a new asynchronous task and passes correction feedback together with the previous plan to Agent Engine. |
| `POST /ai/issues/{id}/approve` | Changes workflow state to `approved` and prepares the generated-files contract. It does not generate code or create a pull request. |
| `POST /ai/issues/{id}/reject` | Changes workflow state to `rejected`. |

## Backend to Agent Engine integration

```text
GitFlame issue
      |
      v
Go backend validation
      |
      v
asynchronous agent task ---- GET /ai/tasks/{taskId}
      |
      v
POST Agent Engine /v1/plans/generate
      |
      v
validated plan.md
      |
      +---- approve -> generated-files contract
      +---- correct -> new task with previous plan + feedback
      +---- reject  -> rejected state
```

The Agent Engine client accepts configuration through `AGENT_ENGINE_URL` and `AGENT_ENGINE_TIMEOUT_SECONDS`. Its request matches the implemented Python service: `request_id`, `issue`, `repository`, raw `configuration_yaml`, `repository_files`, `previous_plan`, and `correction_feedback`. The completed task exposes relevant files, model ID, bounded usage metrics, and a short tool-execution summary without model reasoning.

Supported downstream errors `400`, `404`, `422`, `502`, `503`, and `504` are preserved in the failed task payload. Connection errors become `502`; timeouts become `504`.

The infrastructure configuration provides Redis 7 through `REDIS_URL=redis://redis:6379/0`, with AOF persistence, a health check, and a named data volume. The backend publishes idempotent jobs to Redis Streams, and the standalone Agent Worker consumes them through a consumer group with concurrency `1`. Temporary `502`, `503`, and `504` failures are retried; exhausted and permanent failures are written to a dead-letter stream.

PostgreSQL is the authoritative store for issue sessions, plan revisions, correction feedback, agent task attempts and statuses, generated plans, bounded usage metadata, and recommendation retention. Redis is transport only.

## Verification

- OpenAPI: `GET /openapi.json`
- Swagger UI: `GET /swagger/`
- Contract test: `internal/agent/client_test.go`
- End-to-end HTTP workflow test: `internal/httpapi/integration_test.go`
- Verified locally with `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./cmd/server`.

PR and Issue links should be added here after the branch is published to GitFlame/GitHub.
