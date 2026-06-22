# S0 Acceptance Harness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the interactive S0 smoke script with one acceptance harness that supports local diagnostics and non-interactive CI gating.

**Architecture:** A thin Bash entry point validates command-line options and invokes a Python standard-library test runner. The Python runner performs HTTP and MySQL assertions, writes JSON/Markdown/JUnit reports, and returns a non-zero exit code whenever an S0 gate fails. Existing production defects are reported by the harness rather than hidden or automatically repaired.

**Tech Stack:** Bash, Python 3 standard library, curl-compatible HTTP APIs, MySQL CLI

---

### Task 1: Acceptance result model and reports

**Files:**
- Create: `backend/scripts/s0_acceptance.py`
- Create: `backend/scripts/test_s0_acceptance.py`

1. Write failing unit tests for pass/fail aggregation, gate summaries, JSON, Markdown, and JUnit output.
2. Run `python3 -m unittest backend/scripts/test_s0_acceptance.py` and confirm failure.
3. Implement the result model and report writers.
4. Re-run the tests and confirm they pass.

### Task 2: HTTP and MySQL acceptance gates

**Files:**
- Modify: `backend/scripts/s0_acceptance.py`
- Modify: `backend/scripts/test_s0_acceptance.py`

1. Write failing tests for JSON HTTP handling, schema assertions, E2E metadata assertions, no-fabrication checks, and feedback persistence checks.
2. Implement injectable HTTP and MySQL adapters using only the Python standard library and the `mysql` CLI.
3. Add gates for environment health, schema compatibility, Python endpoint contracts, full intent→tool→answer flow, no-fabrication behavior, decision-log persistence, and feedback.
4. Re-run unit tests.

### Task 3: LLM intent quality gate

**Files:**
- Modify: `backend/scripts/s0_acceptance.py`
- Modify: `services/agent/test_intent_eval.py`

1. Add an acceptance gate that loads `services/agent/test_intent_cases.json`, calls the real sidecar, and requires at least 85% accuracy.
2. Make the standalone intent evaluator return exit code 1 when the sidecar is unavailable or accuracy is below threshold.
3. Verify evaluator syntax and harness unit tests.

### Task 4: Local and CI shell entry point

**Files:**
- Modify: `backend/scripts/s0_verify.sh`
- Create: `backend/scripts/test_s0_verify.sh`

1. Write shell tests for `--help`, invalid modes, local defaults, CI defaults, and exit-code propagation.
2. Replace interactive prompts with `--mode local|ci`.
3. Support `--skip-intent-eval` for fast local diagnostics while keeping intent evaluation mandatory by default.
4. Verify with `bash -n` and shell tests.

### Task 5: Documentation and final verification

**Files:**
- Modify: `docs/testing/s0-verification-guide.html`

1. Document S0 scope and distinguish S1 concerns such as session state and inventory credibility presentation.
2. Document required environment variables, commands, gates, reports, and exit behavior.
3. Run Python unit tests, shell tests, Bash syntax checks, and Go tests.
4. Run the harness against the current local services when available and report genuine S0 failures without masking them.

