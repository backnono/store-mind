# Customer QA LLM + RAG Orchestrator Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade the existing `customerqa` chat flow so LLM handles intent understanding and answer composition, RAG handles non-structured knowledge recall, and existing backend tools continue serving strong factual store data.

**Architecture:** Keep the current Go backend as the single runtime. Introduce an application-layer orchestrator with replaceable `IntentAnalyzer`, `AnswerComposer`, and `Retriever` interfaces. Preserve the current rule-based chat flow as an explicit fallback while adding new decision and retrieval logging around tool and knowledge evidence.

**Tech Stack:** Go, Gin, existing `customerqa` application/domain layers, MySQL-backed repositories, OpenAI-compatible LLM adapter, lightweight keyword-based retrieval in v1, `go test`.

---

### Task 1: Freeze current chat behavior with tests

**Files:**
- Modify: `backend/application/customerqa/service_test.go`
- Modify: `backend/api/http/handler_customer_qa_test.go`

**Step 1: Write missing failing tests for compatibility**

Add tests that assert:

- `Chat` still returns `session_id`, `message_id`, `intent`, `answer`, `cards`, `handoff_required`
- existing inventory/location/promotion/faq behavior remains valid when fallback is used
- handler `meta.request_id` remains present

**Step 2: Run targeted tests to verify baseline**

Run: `go test ./backend/application/customerqa ./backend/api/http`
Expected: PASS on current implementation before refactor

**Step 3: Commit**

```bash
git add backend/application/customerqa/service_test.go backend/api/http/handler_customer_qa_test.go
git commit -m "test: freeze customer qa chat api behavior"
```

### Task 2: Introduce orchestrator interfaces and result model

**Files:**
- Create: `backend/application/customerqa/orchestrator.go`
- Create: `backend/application/customerqa/llm.go`
- Create: `backend/application/customerqa/retriever.go`
- Modify: `backend/application/customerqa/service.go`

**Step 1: Write failing orchestrator unit tests**

Create tests for:

- tool route selected for inventory question
- rag route selected for FAQ question
- hybrid route selected for mixed question
- fallback chosen when analyzer errors

**Step 2: Run tests to verify failure**

Run: `go test ./backend/application/customerqa`
Expected: FAIL because orchestrator types do not exist

**Step 3: Add minimal interfaces and types**

Define:

- `type Orchestrator interface`
- `type IntentAnalyzer interface`
- `type AnswerComposer interface`
- `type Retriever interface`
- `type Decision struct`
- `type Evidence struct`
- `type OrchestratorResult struct`

Update `service` to accept an orchestrator dependency while still supporting a fallback implementation.

**Step 4: Run tests**

Run: `go test ./backend/application/customerqa`
Expected: PASS with minimal stubs and existing behavior intact

**Step 5: Commit**

```bash
git add backend/application/customerqa/orchestrator.go backend/application/customerqa/llm.go backend/application/customerqa/retriever.go backend/application/customerqa/service.go
git commit -m "feat: add customer qa orchestrator interfaces"
```

### Task 3: Move current rule logic into an explicit fallback orchestrator

**Files:**
- Create: `backend/application/customerqa/fallback.go`
- Modify: `backend/application/customerqa/service.go`
- Modify: `backend/application/customerqa/service_test.go`

**Step 1: Write failing tests for fallback orchestration**

Add tests that assert:

- rule-based intent routing still works through orchestrator path
- fallback records route and fallback usage
- unsupported questions still return safe default guidance

**Step 2: Run tests to verify failure**

Run: `go test ./backend/application/customerqa`
Expected: FAIL because fallback orchestrator is not wired

**Step 3: Implement fallback orchestrator**

Move current `routeIntent`, `answerChat`, and related helper behavior into a `fallbackOrchestrator` that returns a normalized orchestrator result.

**Step 4: Run tests**

Run: `go test ./backend/application/customerqa`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/application/customerqa/fallback.go backend/application/customerqa/service.go backend/application/customerqa/service_test.go
git commit -m "refactor: isolate customer qa fallback orchestrator"
```

### Task 4: Add decision metadata to chat response

**Files:**
- Modify: `backend/application/customerqa/service.go`
- Modify: `backend/api/http/handler_customer_qa.go`
- Modify: `backend/api/http/handler_customer_qa_test.go`

**Step 1: Write failing tests for response metadata**

Add assertions for:

- `meta.request_id` still exists
- optional `meta.route`, `meta.confidence`, `meta.rewrite_query`, `meta.fallback_used` are returned when available

**Step 2: Run tests to verify failure**

Run: `go test ./backend/api/http`
Expected: FAIL because new metadata is not mapped

**Step 3: Implement response metadata plumbing**

Extend application response model and handler JSON mapping without breaking old fields.

**Step 4: Run tests**

Run: `go test ./backend/api/http ./backend/application/customerqa`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/application/customerqa/service.go backend/api/http/handler_customer_qa.go backend/api/http/handler_customer_qa_test.go
git commit -m "feat: expose customer qa routing metadata"
```

### Task 5: Add domain contracts for knowledge retrieval and chat decision logging

**Files:**
- Modify: `backend/domain/customerqa/repository.go`
- Modify: `backend/domain/customerqa/entity.go`
- Create: `backend/domain/customerqa/knowledge.go`
- Modify: `backend/application/customerqa/service_test.go`

**Step 1: Write failing tests for new contracts**

Add tests that expect:

- retrieval evidence can be represented in application results
- decision log persistence is attempted when available

**Step 2: Run tests to verify failure**

Run: `go test ./backend/application/customerqa ./backend/domain/customerqa`
Expected: FAIL because types and interfaces do not exist

**Step 3: Implement minimal domain additions**

Add:

- knowledge document or chunk entity
- retrieval repository contract
- decision log entity or equivalent persistence contract

Keep additions minimal and scoped to chat orchestration.

**Step 4: Run tests**

Run: `go test ./backend/application/customerqa ./backend/domain/customerqa`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/domain/customerqa/repository.go backend/domain/customerqa/entity.go backend/domain/customerqa/knowledge.go backend/application/customerqa/service_test.go
git commit -m "feat: add customer qa knowledge retrieval contracts"
```

### Task 6: Implement lightweight MySQL retriever for FAQ and store policy

**Files:**
- Create: `backend/infra/retrieval/mysql_retriever.go`
- Create: `backend/infra/retrieval/mysql_retriever_test.go`
- Modify: `backend/infra/persistence/mysql/repository_customer_qa.go`

**Step 1: Write failing retrieval tests**

Cover:

- FAQ keyword hit
- store policy topic hit
- empty result when no matching knowledge exists
- result ranking returns top relevant chunks first

**Step 2: Run tests to verify failure**

Run: `go test ./backend/infra/retrieval`
Expected: FAIL because retriever is not implemented

**Step 3: Implement retriever**

Use current FAQ data first. If policy data is not yet modeled, stub the contract with FAQ-only retrieval and mark policy hooks clearly.

**Step 4: Run tests**

Run: `go test ./backend/infra/retrieval ./backend/infra/persistence/mysql`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/infra/retrieval/mysql_retriever.go backend/infra/retrieval/mysql_retriever_test.go backend/infra/persistence/mysql/repository_customer_qa.go
git commit -m "feat: add customer qa mysql retriever"
```

### Task 7: Add LLM adapter interfaces with deterministic fake implementation

**Files:**
- Create: `backend/infra/ai/fake_client.go`
- Create: `backend/infra/ai/fake_client_test.go`
- Modify: `backend/application/customerqa/orchestrator.go`
- Modify: `backend/application/customerqa/service_test.go`

**Step 1: Write failing tests for analyzer and composer integration**

Add tests that assert:

- analyzer output controls route selection
- composer answer uses provided evidence
- analyzer failure triggers fallback

**Step 2: Run tests to verify failure**

Run: `go test ./backend/application/customerqa ./backend/infra/ai`
Expected: FAIL because fake LLM adapter is missing

**Step 3: Implement deterministic fake adapter**

Provide a fake analyzer and fake composer suitable for unit tests without network access.

**Step 4: Run tests**

Run: `go test ./backend/application/customerqa ./backend/infra/ai`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/infra/ai/fake_client.go backend/infra/ai/fake_client_test.go backend/application/customerqa/orchestrator.go backend/application/customerqa/service_test.go
git commit -m "test: add deterministic customer qa llm adapters"
```

### Task 8: Implement primary orchestrator flow for tool, rag, and hybrid routes

**Files:**
- Modify: `backend/application/customerqa/orchestrator.go`
- Modify: `backend/application/customerqa/service.go`
- Modify: `backend/application/customerqa/service_test.go`

**Step 1: Write failing behavior tests**

Cover:

- inventory question uses tool evidence and composed answer
- FAQ question uses retrieval evidence and composed answer
- mixed question uses both evidence types
- empty evidence returns conservative answer

**Step 2: Run tests to verify failure**

Run: `go test ./backend/application/customerqa`
Expected: FAIL because orchestrator flow is incomplete

**Step 3: Implement orchestrator**

Implement:

- analyzer call
- route normalization
- evidence collection
- composer call
- fallback on analyzer/composer/retriever/tool failure

**Step 4: Run tests**

Run: `go test ./backend/application/customerqa`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/application/customerqa/orchestrator.go backend/application/customerqa/service.go backend/application/customerqa/service_test.go
git commit -m "feat: implement customer qa orchestrator flow"
```

### Task 9: Wire infrastructure dependencies in bootstrap

**Files:**
- Modify: `backend/internal/bootstrap/app.go`
- Modify: `backend/api/http/handler_customer_qa_test.go`
- Modify: `backend/application/customerqa/service.go`

**Step 1: Write failing wiring test or smoke assertion**

Add a test or constructor assertion that the default service wiring uses:

- primary orchestrator when analyzer and retriever exist
- fallback orchestrator when optional AI dependencies are not configured

**Step 2: Run tests to verify failure**

Run: `go test ./backend/internal/bootstrap ./backend/api/http`
Expected: FAIL because new wiring is not present

**Step 3: Implement wiring**

Keep the system bootable without external AI configuration by defaulting to fallback components.

**Step 4: Run tests**

Run: `go test ./backend/internal/bootstrap ./backend/api/http ./backend/application/customerqa`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/bootstrap/app.go backend/api/http/handler_customer_qa_test.go backend/application/customerqa/service.go
git commit -m "feat: wire customer qa orchestrator dependencies"
```

### Task 10: Verify end-to-end backend test suite

**Files:**
- Modify: `docs/plans/2026-06-02-customer-qa-llm-rag-design.md`
- Modify: `docs/plans/2026-06-02-customer-qa-llm-rag-implementation-plan.md`

**Step 1: Run focused verification**

Run: `go test ./backend/...`
Expected: PASS

**Step 2: Record any deviations**

If some integrations remain stubbed, update the design and plan docs with exact limitations instead of leaving them implicit.

**Step 3: Commit**

```bash
git add docs/plans/2026-06-02-customer-qa-llm-rag-design.md docs/plans/2026-06-02-customer-qa-llm-rag-implementation-plan.md
git commit -m "docs: finalize customer qa llm rag plan"
```

---

## Verification Result

Completed on the current branch with:

```bash
/opt/homebrew/opt/go@1.24/bin/go test ./...
```

Observed result:

- `ok   store-mind/api/http`
- `ok   store-mind/application/customerqa`
- `ok   store-mind/infra/ai`
- `ok   store-mind/infra/logger`
- `ok   store-mind/infra/persistence/mysql`
- `ok   store-mind/infra/retrieval`
- `ok   store-mind/internal/bootstrap`

## Recorded Deviations

The implementation is structurally complete for this phase, with these intentional limitations still present:

- Runtime bootstrap remains conservative.
  - `internal/bootstrap.Build()` still constructs the fallback-safe service by default.
  - Primary orchestrator dependency wiring is implemented through helper construction and covered by tests, but not enabled automatically in production bootstrap.
- AI integration is test-only for now.
  - `infra/ai/fake_client.go` provides deterministic fake analyzer and composer behavior.
  - No real external LLM client is wired yet.
- Retrieval is FAQ-first.
  - `store_policy` retrieval currently reuses FAQ category mapping.
  - Dedicated policy documents and `product_knowledge` sources are not implemented yet.
- Logging is partial.
  - Decision log persistence is supported when the repository implements the optional contract.
  - Retrieval logs and answer logs are not implemented yet.
