import json
from typing import Any


class ContextCompressor:
    """Bound oversized untrusted context before it enters the model conversation."""

    def __init__(self, context_limit_tokens: int, max_tool_output_chars: int) -> None:
        # A conservative token approximation leaves space for output and provider overhead.
        self.max_context_chars = max(8_000, context_limit_tokens * 3)
        self.max_tool_output_chars = max_tool_output_chars

    def compress_text(self, value: str, limit: int) -> str:
        if len(value) <= limit:
            return value
        marker = "\n...[truncated by context compression]...\n"
        remaining = max(0, limit - len(marker))
        head = remaining * 2 // 3
        tail = remaining - head
        return value[:head] + marker + value[-tail:]

    def compress_tool_result(self, result: Any) -> str:
        serialized = json.dumps(result, ensure_ascii=False, separators=(",", ":"))
        return self.compress_text(serialized, self.max_tool_output_chars)

    def fit_messages(self, messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
        total = sum(len(str(message.get("content") or "")) for message in messages)
        if total <= self.max_context_chars:
            return messages

        preserved = [dict(message) for message in messages]
        budget = self.max_context_chars
        for message in reversed(preserved):
            content = str(message.get("content") or "")
            if len(content) <= budget:
                budget -= len(content)
                continue
            message["content"] = self.compress_text(content, max(512, budget))
            budget = 0
        return preserved
