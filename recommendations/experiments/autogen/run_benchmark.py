import argparse
import asyncio
import json
import platform
import resource
import statistics
import time
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from agent_engine.models import GeneratePlanRequest
from agent_engine.service import AgentEngineService
from agent_engine.settings import AgentSettings

ROOT = Path(__file__).resolve().parents[2]


def percentile(values: list[float], fraction: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, round((len(ordered) - 1) * fraction)))
    return ordered[index]


def peak_rss_mb() -> float:
    value = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    # macOS reports bytes; Linux reports KiB.
    divisor = 1024 * 1024 if platform.system() == "Darwin" else 1024
    return value / divisor


def summarize(runs: list[dict[str, Any]]) -> dict[str, Any]:
    successful = [run for run in runs if run["status"] == "ok"]
    latencies = [run["latency_seconds"] for run in successful]
    generation_times = [run["generation_time_seconds"] for run in successful]
    tool_calls = [run["tool_calls"] for run in successful]
    return {
        "attempted": len(runs),
        "successful": len(successful),
        "success_rate": len(successful) / len(runs) if runs else 0.0,
        "latency_p50_seconds": percentile(latencies, 0.50),
        "latency_p95_seconds": percentile(latencies, 0.95),
        "average_generation_time_seconds": (
            statistics.mean(generation_times) if generation_times else None
        ),
        "average_tool_calls": statistics.mean(tool_calls) if tool_calls else None,
        "peak_agent_process_rss_mb": max(
            (run["agent_process_peak_rss_mb"] for run in runs), default=None
        ),
        "model_runtime_memory_mb": None,
        "model_runtime_memory_note": (
            "The OpenAI-compatible contract does not expose runtime memory. Record the model "
            "server/GPU metric on the demo stand alongside this benchmark."
        ),
    }


async def run_case(
    service: AgentEngineService,
    fixture_path: Path,
    repetition: int,
) -> dict[str, Any]:
    request = GeneratePlanRequest.model_validate_json(fixture_path.read_text())
    started = time.perf_counter()
    try:
        response = await service.generate(request)
        latency = time.perf_counter() - started
        return {
            "fixture": fixture_path.name,
            "repetition": repetition,
            "status": "ok",
            "latency_seconds": latency,
            "generation_time_seconds": response.usage.generation_time_seconds,
            "prompt_tokens": response.usage.prompt_tokens,
            "completion_tokens": response.usage.completion_tokens,
            "tool_calls": response.usage.tool_calls,
            "agent_process_peak_rss_mb": peak_rss_mb(),
            "plan_markdown": response.plan_markdown,
        }
    except Exception as exc:
        return {
            "fixture": fixture_path.name,
            "repetition": repetition,
            "status": "error",
            "latency_seconds": time.perf_counter() - started,
            "generation_time_seconds": 0.0,
            "tool_calls": 0,
            "agent_process_peak_rss_mb": peak_rss_mb(),
            "error": f"{type(exc).__name__}: {exc}",
        }


def render_markdown(payload: dict[str, Any]) -> str:
    summary = payload["summary"]
    rows = [
        "| Fixture | Run | Status | Latency, s | Generation, s | Tool calls |",
        "| --- | ---: | --- | ---: | ---: | ---: |",
    ]
    for run in payload["runs"]:
        rows.append(
            f"| {run['fixture']} | {run['repetition']} | {run['status']} | "
            f"{run['latency_seconds']:.3f} | {run['generation_time_seconds']:.3f} | "
            f"{run['tool_calls']} |"
        )
    return "\n".join(
        [
            "# Agent Engine Benchmark",
            "",
            f"- Model: `{payload['environment']['model']}`",
            f"- Endpoint: `{payload['environment']['openai_base_url']}`",
            f"- Quantization: `{payload['environment']['quantization']}`",
            f"- Attempts: {summary['attempted']}",
            f"- Success rate: {summary['success_rate']:.1%}",
            f"- P50 latency: {_metric(summary['latency_p50_seconds'])}",
            f"- P95 latency: {_metric(summary['latency_p95_seconds'])}",
            f"- Average generation time: {_metric(summary['average_generation_time_seconds'])}",
            f"- Peak Agent Engine RSS: {_metric(summary['peak_agent_process_rss_mb'], ' MB')}",
            "- Model runtime memory: provider metric unavailable; capture it on the demo "
            "GPU stand.",
            "",
            *rows,
            "",
            "Errors are recorded without mock or rule-based fallback.",
            "",
        ]
    )


def _metric(value: float | None, suffix: str = " s") -> str:
    return "N/A" if value is None else f"{value:.3f}{suffix}"


async def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixtures", type=Path, default=ROOT / "experiments/autogen/fixtures")
    parser.add_argument(
        "--output-json",
        type=Path,
        default=ROOT / "experiments/autogen/results/benchmark.json",
    )
    parser.add_argument(
        "--output-markdown",
        type=Path,
        default=ROOT / "experiments/autogen/results/benchmark.md",
    )
    parser.add_argument("--model", default=AgentSettings.model)
    parser.add_argument("--base-url", default=AgentSettings.openai_base_url)
    parser.add_argument("--api-key")
    parser.add_argument("--quantization", default="TBD")
    parser.add_argument("--repetitions", type=int, default=1)
    parser.add_argument("--timeout", type=float, default=600)
    args = parser.parse_args()

    settings = AgentSettings(
        model=args.model,
        openai_base_url=args.base_url.rstrip("/"),
        openai_api_key=args.api_key,
        request_timeout_seconds=args.timeout,
        agent_timeout_seconds=args.timeout,
    )
    service = AgentEngineService(settings)
    fixtures = sorted(args.fixtures.glob("*.json"))
    runs = []
    for repetition in range(1, args.repetitions + 1):
        for fixture in fixtures:
            print(f"Running {fixture.name} ({repetition}/{args.repetitions})", flush=True)
            runs.append(await run_case(service, fixture, repetition))

    payload = {
        "benchmark_version": 2,
        "created_at": datetime.now(UTC).isoformat(),
        "environment": {
            "platform": platform.platform(),
            "python": platform.python_version(),
            "model": args.model,
            "quantization": args.quantization,
            "openai_base_url": args.base_url,
            "agent_max_steps": settings.max_steps,
            "agent_max_tool_calls": settings.max_tool_calls,
        },
        "summary": summarize(runs),
        "runs": runs,
    }
    args.output_json.parent.mkdir(parents=True, exist_ok=True)
    args.output_markdown.parent.mkdir(parents=True, exist_ok=True)
    args.output_json.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n")
    args.output_markdown.write_text(render_markdown(payload))


if __name__ == "__main__":
    asyncio.run(main())
