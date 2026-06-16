from dataclasses import dataclass
from typing import Any

import httpx
from pydantic import ValidationError

from recommendation_service.models import RecommendationResponse
from recommendation_service.settings import Settings


class ModelUnavailableError(RuntimeError):
    pass


class ModelTimeoutError(RuntimeError):
    pass


class ModelOutputError(RuntimeError):
    pass


@dataclass(frozen=True)
class InferenceMetrics:
    total_duration_ns: int | None = None
    load_duration_ns: int | None = None
    prompt_eval_count: int | None = None
    prompt_eval_duration_ns: int | None = None
    eval_count: int | None = None
    eval_duration_ns: int | None = None


class OllamaClient:
    def __init__(self, settings: Settings, client: httpx.AsyncClient | None = None) -> None:
        self.settings = settings
        self._client = client

    async def ready(self) -> bool:
        try:
            data = await self._request("GET", "/api/tags")
        except (ModelUnavailableError, ModelTimeoutError, ModelOutputError):
            return False
        names = {model.get("name") for model in data.get("models", [])}
        return self.settings.model in names

    async def analyze(
        self,
        *,
        system_prompt: str,
        user_prompt: str,
        response_schema: dict[str, Any],
    ) -> tuple[RecommendationResponse, InferenceMetrics]:
        payload = {
            "model": self.settings.model,
            "messages": [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ],
            "stream": False,
            "format": response_schema,
            "options": {
                "temperature": 0,
                "seed": self.settings.seed,
                "num_ctx": self.settings.context_tokens,
            },
        }
        data = await self._request("POST", "/api/chat", json=payload)
        try:
            content = data["message"]["content"]
            response = RecommendationResponse.model_validate_json(content)
        except (KeyError, TypeError, ValidationError, ValueError) as exc:
            raise ModelOutputError(f"model returned invalid structured output: {exc}") from exc

        metrics = InferenceMetrics(
            total_duration_ns=data.get("total_duration"),
            load_duration_ns=data.get("load_duration"),
            prompt_eval_count=data.get("prompt_eval_count"),
            prompt_eval_duration_ns=data.get("prompt_eval_duration"),
            eval_count=data.get("eval_count"),
            eval_duration_ns=data.get("eval_duration"),
        )
        return response, metrics

    async def _request(self, method: str, path: str, **kwargs: Any) -> dict[str, Any]:
        owns_client = self._client is None
        client = self._client or httpx.AsyncClient(
            base_url=self.settings.ollama_base_url,
            timeout=self.settings.request_timeout_seconds,
        )
        try:
            response = await client.request(method, path, **kwargs)
            if response.status_code == 404 or response.status_code >= 500:
                raise ModelUnavailableError(
                    f"Ollama returned {response.status_code}: {response.text[:300]}"
                )
            if response.status_code >= 400:
                raise ModelOutputError(
                    f"Ollama rejected the request with {response.status_code}: "
                    f"{response.text[:300]}"
                )
            data = response.json()
            if not isinstance(data, dict):
                raise ModelOutputError("Ollama returned a non-object response")
            return data
        except httpx.TimeoutException as exc:
            raise ModelTimeoutError("model inference timed out") from exc
        except httpx.RequestError as exc:
            raise ModelUnavailableError(f"cannot reach Ollama: {exc}") from exc
        except ValueError as exc:
            raise ModelOutputError("Ollama returned invalid JSON") from exc
        finally:
            if owns_client:
                await client.aclose()
