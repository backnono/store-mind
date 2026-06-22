# LLM Intent Timeout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Prevent successful but slightly slow intent-analysis calls from being canceled prematurely.

**Architecture:** Increase only the intent-analysis timeout at each HTTP boundary. Python gets 5 seconds for DeepSeek, while Go gets 8 seconds for the complete Python sidecar request.

**Tech Stack:** Go `context`/`net/http`, Python `asyncio`, standard test tooling

---

### Task 1: Go intent request timeout

**Files:**
- Modify: `backend/infra/ai/python_llm_client.go`
- Create: `backend/infra/ai/python_llm_client_test.go`

1. Write a failing test asserting the intent request timeout is 8 seconds.
2. Run the focused Go test and confirm it fails.
3. Add an `intentRequestTimeout` constant and use it in `AnalyzeIntent`.
4. Run the focused Go test and confirm it passes.

### Task 2: Python upstream intent timeout

**Files:**
- Modify: `services/agent/intent_analyzer.py`
- Create: `services/agent/test_intent_analyzer.py`

1. Write a failing test asserting `IntentAnalyzer.analyze` passes 5 seconds to `asyncio.wait_for`.
2. Run the focused Python test and confirm it fails.
3. Add an `INTENT_LLM_TIMEOUT_SECONDS` constant and use it in `analyze`.
4. Run the focused Python test and confirm it passes.

### Task 3: Verification

1. Run Go tests from `backend/`.
2. Run the focused Python unit test from `services/agent/`.
3. Review the diff to ensure unrelated working-tree changes were preserved.

