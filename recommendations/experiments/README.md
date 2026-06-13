# ML Experimentation Setup

## Goal

Compare real open-source code models on the GitFlame recommendation contract under identical
conditions, then select the most suitable Sprint 1 candidate.

## Environment

- Apple MacBook Pro, M3 Pro, 18 GB unified memory;
- Ollama quantized runtime;
- fixed prompt and JSON Schema;
- `temperature=0`, seed `42`;
- three manually labelled synthetic repositories covering all five recommendation categories.

No private repository content is used.

## Models

```bash
ollama pull qwen2.5-coder:1.5b
ollama pull qwen2.5-coder:7b
ollama pull deepseek-coder:1.3b
ollama pull codellama:7b-instruct
```

## Run

```bash
uv sync --dev
uv run pytest
uv run python experiments/run_benchmark.py
```

Run only the selected baseline:

```bash
uv run python experiments/run_benchmark.py --models qwen2.5-coder:1.5b
```

Raw results are written to `experiments/results/model_benchmark.json`. The benchmark records model
errors rather than replacing them with fake recommendations.

## Fixtures And Labels

| Fixture | Categories |
| --- | --- |
| `fixtures/web_api.json` | security, performance |
| `fixtures/maintainability.json` | code duplication, maintainability |
| `fixtures/architecture.json` | architecture, security |

Each expected finding contains category, file, line, and severity. A predicted finding is counted as
a true positive when category and file match and the reported line is within three lines of the
manual label.

## Metrics

- finding precision, recall, and F1;
- schema-valid rate;
- file/line-valid rate;
- category coverage;
- mean wall-clock latency;
- Ollama output tokens per second;
- recorded Ollama load and inference durations;
- manual review of suggestion usefulness.

Retrieval metrics such as `precision@k`, `recall@k`, `nDCG@k`, and `MRR` are not used here because
Sprint 1 does not yet retrieve top-k vector-search chunks.

## Manual Suggestion Review Rubric

Score each matched suggestion from 0 to 2:

- `0`: incorrect, unsafe, or not actionable;
- `1`: directionally useful but incomplete;
- `2`: correct and actionable for the referenced evidence.

