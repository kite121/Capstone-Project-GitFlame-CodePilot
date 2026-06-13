# Initial Baseline Result

## Status

Completed on June 13, 2026, with the quantized Ollama `qwen2.5-coder:1.5b` model on an Apple M3 Pro
with 18 GB unified memory.

Unlike a rule-based mock, this baseline exercises the complete model-backed flow:

`.yml -> filtered repo_context -> injection-resistant prompt -> Qwen -> structured JSON -> strict validation`

## Selection Hypothesis And Decision

Qwen2.5-Coder-1.5B-Instruct was expected to provide the best Sprint 1 trade-off between supported
code findings, structured-output reliability, latency, model size, and Apache-2.0 licensing. The
benchmark supports selecting it as the deployment baseline: Qwen 1.5B tied Qwen 7B on automatic
F1, was substantially faster and smaller, and kept 100% schema and file/line validity.

## Manual Analysis Ground Truth

The three synthetic fixtures contain six labelled findings across security, performance,
code duplication, maintainability, and architecture. The benchmark compares model findings with
those labels using category/file equality and a three-line tolerance.

## Measured Result

| Metric | Qwen2.5-Coder-1.5B result |
| --- | ---: |
| Schema-valid rate | 100% |
| File/line-valid rate | 100% |
| Finding precision | 0.500 |
| Finding recall | 0.333 |
| Finding F1 | 0.400 |
| Mean case latency | 5.28 seconds |
| Output throughput | 96.97 tokens/second |
| Mean reported load duration | 0.55 seconds |
| Ollama model size | 986 MB |
| Maximum loaded size | 1.89 GB |

The model returned four recommendations across the three fixtures. It correctly identified the
hardcoded API token and suggested environment-based secret handling. It did not identify the
labelled SQL injection, architecture, or maintainability findings. It produced an unsupported
duplication claim in `src/order.py` and a performance finding with an ineffective connection-pool
remediation.

## Manual Suggestion Review

The two automatically matched Qwen 1.5B findings averaged `1/2` under the manual usefulness rubric:

- the hardcoded-token finding received `2/2` because its evidence and remediation were correct;
- the matched cross-file duplication finding received `0/2` because it incorrectly described
  duplication between `order.py` and `slack_report.py`.

This exposes a limitation of category/file/line-tolerance matching: an automatic true positive can
still be semantically wrong.

## Baseline Conclusion

The baseline proves the real end-to-end ML integration and strict failure contract. It is suitable
for a Sprint 1 demonstration and a public synthetic-data Space. Its measured recall and
hallucination rate are not sufficient for production use. The next quality steps are stronger
prompt/evaluation iterations, more representative labelled repositories, and the planned
repository-aware RAG retrieval stage.

Raw evidence and the complete comparison are available at:

- `experiments/results/model_benchmark.json`
- `model_comparison.md`
