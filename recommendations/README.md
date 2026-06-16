---
title: GitFlame CodePilot Recommendations
emoji: 🔥
colorFrom: yellow
colorTo: red
sdk: docker
app_port: 7860
pinned: false
license: apache-2.0
models:
  - Qwen/Qwen2.5-Coder-1.5B-Instruct
---

# GitFlame CodePilot Recommendation ML Service

Sprint 1 deliverables for the external GitFlame AI integration service. This component accepts a
GitFlame-style `.yml` configuration and repository file context, runs a real open-source code model,
and returns a strictly validated recommendation response.

The service has **no rule-based or mock inference fallback**. If Ollama or the configured model is
unavailable, slow, or returns invalid output, the API returns an explicit error.

## Handoff For Roma

| Required report artifact | Path |
| --- | --- |
| ML Experimentation Setup | [`experiments/README.md`](experiments/README.md) |
| Initial Baseline Result | [`experiments/results/initial_baseline_result.md`](experiments/results/initial_baseline_result.md) |
| Raw Model Benchmark | [`experiments/results/model_benchmark.json`](experiments/results/model_benchmark.json) |
| Recommendation Prompt | [`recommendation_prompt.md`](recommendation_prompt.md) |
| Recommendation Schema | [`recommendation_schema.json`](recommendation_schema.json) |
| Model Comparison | [`model_comparison.md`](model_comparison.md) |
| Future RAG / Vector Search | [`rag_vector_search_direction.md`](rag_vector_search_direction.md) |
| Deployment Guide | [`deployment_guide.md`](deployment_guide.md) |
| Public Hugging Face Space | [`KarimKhab/gitflame-codepilot-recommendations`](https://huggingface.co/spaces/KarimKhab/gitflame-codepilot-recommendations) |

### Report-ready summary

The recommendation ML component was implemented as an external FastAPI service because the team
does not modify GitFlame internals. The service applies repository analysis settings from `.yml`,
constructs an injection-resistant prompt with numbered source lines, invokes a locally or remotely
hosted open-source model through Ollama structured outputs, and validates every recommendation
against a strict JSON Schema and the supplied repository context. Qwen2.5-Coder-1.5B-Instruct was
selected as the Sprint 1 candidate because it balances code-analysis capability, Apache-2.0
licensing, 32K context, and low inference requirements. The initial experiment compares it with
Qwen2.5-Coder-7B-Instruct, DeepSeek-Coder-1.3B-Instruct, and CodeLlama-7B-Instruct under the same
quantized Ollama runtime.

## API

| Endpoint | Purpose |
| --- | --- |
| `GET /health` | Confirms the HTTP service is running. |
| `GET /ready` | Confirms the configured Ollama model is available. |
| `POST /v1/recommendations/analyze` | Runs repository recommendation analysis. |
| `GET /docs` | FastAPI OpenAPI UI. |

Request:

```json
{
  "config_yaml": "version: 1\nanalysis:\n  enabled: true\n...",
  "repo_context": [
    {
      "path": "src/app.py",
      "content": "def example():\n    pass\n"
    }
  ]
}
```

Response:

```json
{
  "summary": "Repository-level summary.",
  "recommendations": [
    {
      "severity": "high",
      "category": "security",
      "file": "src/app.py",
      "line": 12,
      "problem": "Evidence-backed problem description.",
      "suggestion": "Actionable remediation.",
      "confidence": 0.96
    }
  ]
}
```

## Local Run

Prerequisites: Python 3.12, `uv`, Ollama, and the Qwen model.

```bash
ollama pull qwen2.5-coder:1.5b
uv sync --dev
uv run uvicorn recommendation_service.app:app --host 0.0.0.0 --port 8000
```

In another terminal:

```bash
curl http://localhost:8000/ready
curl -X POST http://localhost:8000/v1/recommendations/analyze \
  -H 'Content-Type: application/json' \
  --data @experiments/requests/web_api_request.json
```

Run tests:

```bash
uv run pytest
uv run ruff check .
```

The model is configured by the service owner, never by repository `.yml`:

```bash
RECOMMENDATION_MODEL=qwen2.5-coder:7b \
OLLAMA_BASE_URL=http://127.0.0.1:11434 \
uv run uvicorn recommendation_service.app:app --port 8000
```

## Failure Contract

| Status | Meaning |
| --- | --- |
| `422` | Invalid input, YAML, disabled analysis, or empty filtered context. |
| `502` | Model returned malformed output or invalid file/line references. |
| `503` | Ollama or the selected model is unavailable. |
| `504` | Model inference timed out. |

Public deployment must only receive synthetic or explicitly approved repository content.

Live API base URL:

`https://karimkhab-gitflame-codepilot-recommendations.hf.space`
