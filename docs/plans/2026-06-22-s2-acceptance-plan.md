# S2 Acceptance Harness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build one repeatable S2 acceptance workflow for the quality-strengthening scope in `docs/design/gap-analysis/gap-analysis.html`.

**Architecture:** Reuse the S0/S1 pattern: a Bash entry point calls a Python standard-library acceptance runner, which performs HTTP, MySQL, offline dataset, and report-generation checks. Go unit tests continue to verify internal orchestration, while the S2 harness gates user-visible behavior: FAQ semantic retrieval, compound intents, price comparison, Admin CRUD, and full regression.

**Tech Stack:** Bash, Python 3 standard library, Go tests, curl-compatible HTTP APIs, MySQL CLI, existing Go backend and Python LLM sidecar.

---

## Acceptance Scope

S2 is accepted only when these gates pass:

1. FAQ semantic matching hit rate is at least 90% on an offline FAQ paraphrase set.
2. Compound intents are split, routed, and answered as one coherent response.
3. Price query and price comparison work for single-product and multi-product questions.
4. Admin CRUD endpoints under `/api/admin/resources/*` can create, update, and delete every exposed resource type.
5. S0 and S1 critical regression checks still pass.
6. Bad case review output exists for failed or low-confidence cases, even when the final gate passes.

## Test Data

Create a dedicated S2 fixture so the harness does not depend on production-like seed drift.

**Files:**
- Create: `backend/db/seeds/004_s2_test_data.sql`
- Create: `services/agent/test_s2_faq_cases.json`
- Create: `services/agent/test_s2_chat_cases.json`

Minimum fixture coverage:

- FAQ paraphrases: at least 30 questions mapped to canonical FAQ ids, including payment, refund, invoice, membership, delivery, freshness, and customer service.
- Compound chat cases: at least 20 messages covering `product_location,inventory`, `price,inventory`, `price,product_location`, `faq,product_location`, and unsupported mixed questions.
- Price cases: at least 10 messages covering exact product, alias, two-product comparison, unknown product, and equal-price comparison.
- Admin CRUD entities: one isolated `s2_acceptance_*` store tree with zone, shelf, product, SKU, product location, inventory, promotion, and FAQ rows.

## Task 1: S2 Offline Dataset Gates

**Files:**
- Create: `services/agent/test_s2_faq_cases.json`
- Create: `services/agent/test_s2_chat_cases.json`
- Create: `services/agent/test_s2_eval.py`

1. Write the FAQ dataset with fields: `id`, `query`, `expected_faq_id`, `expected_keywords`, `category`.
2. Write the chat dataset with fields: `id`, `message`, `expected_intents`, `expected_route`, `expected_keywords`, `forbidden_keywords`.
3. Implement `test_s2_eval.py` to call `POST /llm/intent` for chat cases and `GET /api/v1/customer-qa/faqs/search` for FAQ cases.
4. Gate FAQ hit rate at `>= 0.90`.
5. Gate compound intent accuracy at `>= 0.85`, while requiring every listed intent to appear in the model decision.
6. Write low-confidence and failed cases to `artifacts/s2/s2-bad-cases.json`.

Run:

```bash
python3 services/agent/test_s2_eval.py \
  --api http://127.0.0.1:8080 \
  --sidecar http://127.0.0.1:9090
```

Expected: exit code `0` only when FAQ hit rate, compound intent accuracy, and no-fabrication checks pass.

## Task 2: S2 Acceptance Runner

**Files:**
- Create: `backend/scripts/s2_acceptance.py`
- Create: `backend/scripts/test_s2_acceptance.py`

1. Write failing unit tests for result aggregation, JSON/Markdown/JUnit output, threshold evaluation, and bad-case serialization.
2. Implement report writers matching S0/S1:
   - `artifacts/s2/s2-report.json`
   - `artifacts/s2/s2-report.md`
   - `artifacts/s2/s2-report.xml`
   - `artifacts/s2/s2-bad-cases.json`
3. Add injectable HTTP and MySQL adapters using only Python standard library plus `mysql` CLI.
4. Add gates:
   - `faq-semantic`: paraphrase set hit rate `>= 90%`.
   - `compound-intent`: visible `/chat` response preserves all expected sub-intents.
   - `price`: price answer contains expected product names and price values.
   - `admin-crud`: all resource create/update/delete flows return expected status codes and persisted data.
   - `regression`: calls S0/S1 harnesses or verifies equivalent critical flows.
5. Unit test each validator without needing live services.

Run:

```bash
python3 -m unittest backend/scripts/test_s2_acceptance.py
```

Expected: all tests pass.

## Task 3: HTTP E2E Gates

**Files:**
- Modify: `backend/scripts/s2_acceptance.py`
- Modify: `backend/scripts/test_s2_acceptance.py`

1. Implement `/chat` helpers that preserve `session_id` across multi-turn checks.
2. Add compound E2E cases:
   - `可乐在哪儿，还有几瓶？` expects location and inventory evidence.
   - `可乐多少钱，还有货吗？` expects price and inventory evidence.
   - `可乐和雪碧哪个便宜？` expects both products and a comparison.
   - `可乐在哪，退款怎么弄？` expects location plus FAQ/refund content.
3. For every response, assert:
   - HTTP 200.
   - `intent` contains every expected sub-intent, or `meta` exposes equivalent structured sub-intents if the response model evolves.
   - `answer` includes required keywords.
   - `answer` excludes forbidden hallucination phrases.
   - `cards` or `evidence` are present when tool-backed facts are required.
4. Add latency recording as non-blocking diagnostics, with a warning in the Markdown report when p95 exceeds 3 seconds.

Expected: compound answers are coherent and do not drop sub-intents.

## Task 4: Admin CRUD Gates

**Files:**
- Modify: `backend/scripts/s2_acceptance.py`
- Modify: `backend/scripts/test_s2_acceptance.py`

1. Add resource specs for:
   - `/api/admin/resources/stores`
   - `/api/admin/resources/zones`
   - `/api/admin/resources/shelves`
   - `/api/admin/resources/products`
   - `/api/admin/resources/skus`
   - `/api/admin/resources/product-locations`
   - `/api/admin/resources/inventories`
   - `/api/admin/resources/promotions`
   - `/api/admin/resources/faqs`
2. For each resource, run create, update, and delete.
3. Verify create returns `201` and an `item.id`.
4. Verify update returns `200` and changed fields.
5. Verify delete returns `200`.
6. Verify persistence with MySQL after create/update when the resource has a stable table mapping.
7. Use `s2_acceptance_*` names and delete in reverse dependency order during cleanup.

Expected: every Admin CRUD route is reachable and mutates the database correctly.

## Task 5: Shell Entry Point

**Files:**
- Create: `backend/scripts/s2_verify.sh`
- Create: `backend/scripts/test_s2_verify.sh`

1. Write shell tests for `--help`, invalid modes, default local mode, CI mode, threshold flags, and exit-code propagation.
2. Implement options:
   - `--mode local|ci`
   - `--api http://127.0.0.1:8080`
   - `--sidecar http://127.0.0.1:9090`
   - `--mysql-host`, `--mysql-port`, `--mysql-user`, `--mysql-password`, `--mysql-database`
   - `--skip-admin-crud` for diagnostics only; CI must not skip it.
   - `--skip-llm-eval` for diagnostics only; CI must not skip it.
3. Make CI mode fail if fixture SQL has not been applied.
4. Ensure reports always write to `artifacts/s2/`.

Run:

```bash
bash -n backend/scripts/s2_verify.sh
bash backend/scripts/test_s2_verify.sh
```

Expected: shell tests pass and invalid invocations return non-zero.

## Task 6: Regression and Documentation

**Files:**
- Create: `docs/testing/s2-verification-guide.html`
- Modify: `docs/design/gap-analysis/gap-analysis.html`

1. Document S2 scope, prerequisites, fixture setup, commands, reports, and pass/fail thresholds.
2. Link `docs/testing/s2-verification-guide.html` from the S2 section in `docs/design/gap-analysis/gap-analysis.html`.
3. Document the regression suite:
   - `bash backend/scripts/s0_verify.sh --mode ci`
   - `bash backend/scripts/s1_verify.sh --mode ci`
   - `bash backend/scripts/s2_verify.sh --mode ci`
   - `go test ./...`
   - `python3 -m unittest backend/scripts/test_s2_acceptance.py`
4. Add a short bad-case review protocol:
   - Review every item in `artifacts/s2/s2-bad-cases.json`.
   - Classify each as data issue, prompt issue, retrieval issue, orchestration issue, or acceptable ambiguity.
   - Do not mark S2 accepted while any high-risk bad case remains unresolved.

Expected: a new engineer can run S2 acceptance from docs without reading implementation history.

## Final Verification Command

After implementation, run from the repository root:

```bash
mysql -u app -papp -h 127.0.0.1 -P 3307 store_mind < backend/db/seeds/004_s2_test_data.sql
python3 -m unittest backend/scripts/test_s2_acceptance.py
bash -n backend/scripts/s2_verify.sh
bash backend/scripts/test_s2_verify.sh
go test ./...
bash backend/scripts/s2_verify.sh --mode ci
```

S2 can be accepted only when the final command exits `0` and `artifacts/s2/s2-report.md` shows all required gates passing.
