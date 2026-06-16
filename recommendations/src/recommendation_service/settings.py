import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    model: str = "qwen2.5-coder:1.5b"
    ollama_base_url: str = "http://127.0.0.1:11434"
    request_timeout_seconds: float = 180.0
    seed: int = 42
    context_tokens: int = 16_384

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            model=os.getenv("RECOMMENDATION_MODEL", cls.model),
            ollama_base_url=os.getenv("OLLAMA_BASE_URL", cls.ollama_base_url).rstrip("/"),
            request_timeout_seconds=float(
                os.getenv("MODEL_REQUEST_TIMEOUT_SECONDS", str(cls.request_timeout_seconds))
            ),
            seed=int(os.getenv("MODEL_SEED", str(cls.seed))),
            context_tokens=int(os.getenv("MODEL_CONTEXT_TOKENS", str(cls.context_tokens))),
        )

