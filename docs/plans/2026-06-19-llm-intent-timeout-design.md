# LLM Intent Timeout Design

## Problem

The Go client cancels `/llm/intent` after 3 seconds, while the Python sidecar can take slightly longer than 3 seconds to receive a DeepSeek response. This causes Go to report `context deadline exceeded` even though Python completes successfully moments later.

## Design

Use nested timeouts with explicit headroom:

- Python → DeepSeek intent request: 5 seconds.
- Go → Python `/llm/intent` request: 8 seconds.

The outer Go timeout is longer so Python has time to finish its upstream request, serialize the result, and send it over localhost. Existing answer composition and anaphora resolution timeouts remain unchanged.

Both values are represented by named constants and covered by focused tests.

