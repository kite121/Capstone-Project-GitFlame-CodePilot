# Sprint 2 Agent Engine - Karim Deliverables

## Version 2 compared with the Sprint 1 mock service

Sprint 1 generated repository recommendation JSON through a single Ollama call. Version 2 adds a
separate stateless issue-to-plan Agent Engine. It can iteratively inspect repository evidence using
bounded read-only tools and external RAG, consume correction feedback, and return a validated
Markdown implementation plan plus usage metrics. It does not generate code or own workflow state.

## SERGE reuse and adaptation

| SERGE component | Version 2 decision |
| --- | --- |
| OpenAI-compatible LLM client | Reused as the design basis; implemented with streaming, retries, request timeout, tool-call assembly, reasoning capture, readiness, and token usage. |
| Prompt system | Adapted from PR review to the `autogen_prompt.md` issue-to-plan contract with prompt-injection boundaries. |
| Agent Loop | Adapted to `issue -> repository tools/RAG -> validated plan.md`. |
| Context compression | Reused as bounded message and tool-output compression. |
| Sandbox | Reused as an allowlisted in-process executor with no shell, write, credential, or GitHub capability. |
| Repository tools | Adapted to supplied GitFlame files: `read_file`, `list_dir`, and `grep`. |
| External RAG | Added as `search_repository(query, top_k, filters)`. |
| Repository source | Added as an abstraction with a supplied-files implementation and a future clone-cache extension point. |
| Context Script | Not used in Sprint 2. Repository-defined scripts are never executed. |
| GitHub client/auth/actions/review publishing | Not used. GitFlame and the Go backend own integration. |

SERGE source reference: <https://github.com/huggingface/serge>. Apache 2.0 attribution files remain a
repository-level deliverable owned by the architecture task; no copied SERGE source file is included
in this implementation.

## Runtime contract

### `POST /v1/plans/generate`

The request contains `request_id`, issue data, repository metadata, operator-safe YAML settings,
GitFlame-supplied repository files, and optional paired `previous_plan` plus
`correction_feedback`. The response contains validated `plan_markdown`, relevant files, configured
model ID, token usage, tool-call count, reasoning character count, and generation time.

### Health endpoints

- `GET /health`: process liveness;
- `GET /ready`: verifies that the exact configured model is listed by the OpenAI-compatible server.

### Error mapping

| Code | HTTP | Meaning |
| --- | ---: | --- |
| `model_unavailable` | 503 | Model server or selected model is unavailable. |
| `rag_unavailable` | 503 | The model requested RAG but the RAG contract is unavailable. |
| `invalid_output` | 502 | Output violates the required plan format or file references. |
| `empty_output` | 502 | Model returned neither content nor tool calls. |
| `tool_limit_exceeded` | 422 | Tool-call or Agent Loop step bound was exhausted. |
| `inference_timeout` | 504 | Per-call or overall Agent Loop timeout expired. |

## Safety and validation

- Issue text, YAML, repository content, previous plans, feedback, and RAG results are untrusted.
- Model selection and credentials are operator-controlled and cannot be supplied in repository YAML.
- Tools are read-only, path-traversal protected, `.git` blocked, range/result bounded, and confined to
  files supplied by GitFlame or the external RAG contract.
- The exact heading order and non-empty sections from `plan_format.md` are required.
- Existing file references must come from supplied or RAG evidence; new files must use `(create)`.
- Hidden reasoning is never returned or stored; only its character count is reported.

## Benchmark and evidence

- Runner: [`experiments/autogen/run_benchmark.py`](experiments/autogen/run_benchmark.py)
- Five issue fixtures: [`experiments/autogen/fixtures`](experiments/autogen/fixtures)
- JSON result: [`experiments/autogen/results/benchmark.json`](experiments/autogen/results/benchmark.json)
- Markdown result: [`experiments/autogen/results/benchmark.md`](experiments/autogen/results/benchmark.md)
- Agent Engine tests: `tests/test_agent_*.py` and `tests/test_plan_validator.py`

The current local Ollama binary crashes before listing models in this environment, so real
quantized-model numbers must be captured on the demo GPU stand with the included runner. The result
files record this blocker instead of presenting fabricated metrics.

## Handoff links

- Agent Engine endpoints: source package `src/agent_engine` and generated `/docs` OpenAPI UI.
- Model artifact: `Qwen/Qwen3-Coder-30B-A3B-Instruct` (quantized artifact link to be inserted after
  demo-stand publication).
- Pull request: to be inserted after this branch is pushed and a PR is created.
- Experiment results: paths listed above; replace the blocked probe with demo-stand results before
  the weekly report is submitted.
