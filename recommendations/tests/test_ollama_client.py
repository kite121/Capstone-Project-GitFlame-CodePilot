import httpx
import pytest

from recommendation_service.models import RecommendationResponse
from recommendation_service.ollama_client import (
    ModelOutputError,
    ModelUnavailableError,
    OllamaClient,
)
from recommendation_service.settings import Settings
from tests.conftest import ollama_response


@pytest.mark.asyncio
async def test_ollama_client_uses_schema_and_parses_metrics():
    captured = {}

    async def handler(request: httpx.Request) -> httpx.Response:
        captured.update(__import__("json").loads(request.content))
        return httpx.Response(
            200,
            json=ollama_response(
                {
                    "summary": "No issues.",
                    "recommendations": [],
                },
                total_duration=100,
                eval_count=10,
                eval_duration=50,
            ),
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://ollama")
    ollama = OllamaClient(Settings(), client=client)

    response, metrics = await ollama.analyze(
        system_prompt="system",
        user_prompt="user",
        response_schema=RecommendationResponse.model_json_schema(),
    )

    assert response.recommendations == []
    assert captured["format"]["additionalProperties"] is False
    assert captured["options"]["temperature"] == 0
    assert metrics.eval_count == 10
    await client.aclose()


@pytest.mark.asyncio
async def test_ollama_client_rejects_invalid_json():
    async def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"message": {"content": "not-json"}})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://ollama")
    ollama = OllamaClient(Settings(), client=client)

    with pytest.raises(ModelOutputError):
        await ollama.analyze(system_prompt="s", user_prompt="u", response_schema={})
    await client.aclose()


@pytest.mark.asyncio
async def test_ollama_client_treats_missing_model_as_unavailable():
    async def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(404, json={"error": "model not found"})

    client = httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://ollama")
    ollama = OllamaClient(Settings(), client=client)

    with pytest.raises(ModelUnavailableError):
        await ollama.analyze(system_prompt="s", user_prompt="u", response_schema={})
    await client.aclose()
