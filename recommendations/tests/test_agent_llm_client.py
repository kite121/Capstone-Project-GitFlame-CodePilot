import json

import httpx
import pytest

from agent_engine.llm_client import OpenAICompatibleClient
from agent_engine.settings import AgentSettings


@pytest.mark.asyncio
async def test_openai_client_parses_streamed_content_reasoning_tools_and_usage():
    captured = {}

    async def handler(request: httpx.Request) -> httpx.Response:
        captured.update(json.loads(request.content))
        events = [
            {"choices": [{"delta": {"reasoning_content": "inspect "}}]},
            {
                "choices": [
                    {
                        "delta": {
                            "tool_calls": [
                                {
                                    "index": 0,
                                    "id": "call-1",
                                    "function": {
                                        "name": "read_file",
                                        "arguments": '{"path":"src/auth.py"}',
                                    },
                                }
                            ]
                        }
                    }
                ]
            },
            {
                "choices": [],
                "usage": {"prompt_tokens": 10, "completion_tokens": 4, "total_tokens": 14},
            },
        ]
        body = "".join(f"data: {json.dumps(event)}\n\n" for event in events) + "data: [DONE]\n\n"
        return httpx.Response(200, content=body, headers={"content-type": "text/event-stream"})

    http_client = httpx.AsyncClient(
        transport=httpx.MockTransport(handler), base_url="http://model/v1"
    )
    client = OpenAICompatibleClient(AgentSettings(), client=http_client)
    result = await client.complete(messages=[{"role": "user", "content": "plan"}], tools=[])

    assert captured["stream"] is True
    assert result.reasoning == "inspect "
    assert result.tool_calls[0].arguments == {"path": "src/auth.py"}
    assert result.usage.total_tokens == 14
    await http_client.aclose()


@pytest.mark.asyncio
async def test_openai_client_ready_checks_exact_model():
    async def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, json={"data": [{"id": AgentSettings.model}]})

    http_client = httpx.AsyncClient(
        transport=httpx.MockTransport(handler), base_url="http://model/v1"
    )
    client = OpenAICompatibleClient(AgentSettings(), client=http_client)

    assert await client.ready() is True
    await http_client.aclose()
