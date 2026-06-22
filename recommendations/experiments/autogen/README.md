# Issue-to-Plan Benchmark

This benchmark exercises the complete Sprint 2 Agent Engine against five issue fixtures. It records
P50/P95 end-to-end latency, average model generation time, tool calls, generation success rate, and
peak Agent Engine process RSS. Failed model calls remain failures; no mock inference fallback is
used.

## Run on the quantized demo stand

Start an OpenAI-compatible server for the selected quantized model, then run:

```bash
uv run python experiments/autogen/run_benchmark.py \
  --model Qwen/Qwen3-Coder-30B-A3B-Instruct \
  --base-url http://127.0.0.1:8000/v1 \
  --quantization AWQ \
  --repetitions 3
```

The runner writes:

- `results/benchmark.json`: environment, raw runs, errors, plans, and aggregate metrics;
- `results/benchmark.md`: report-ready metric table.

The standard OpenAI API does not expose model-server memory. Record GPU/model runtime memory from
the demo server (`nvidia-smi`, vLLM metrics, or the deployment platform) beside the Agent Engine RSS
reported here.

## Acceptance checks

- all five fixtures return a contract-valid `plan.md`;
- every existing file reference is backed by supplied or RAG evidence;
- success rate, P50/P95 latency, generation time, tool calls, and memory evidence are present;
- errors are explicit and no generated plan is substituted after a failure.
