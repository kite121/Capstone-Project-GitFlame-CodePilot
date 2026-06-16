#!/bin/sh
set -eu

ollama serve &
OLLAMA_PID=$!
trap 'kill "${OLLAMA_PID}" 2>/dev/null || true' EXIT

until ollama list >/dev/null 2>&1; do
  sleep 1
done

ollama pull "${RECOMMENDATION_MODEL}"
uvicorn recommendation_service.app:app --host 0.0.0.0 --port 7860
