# Customer QA LLM + RAG Orchestrator Design

Date: 2026-06-02

## Scope

Evolve the existing `backend/application/customerqa` chat flow from rule-based routing and template answers into an orchestrated `LLM + RAG + tools` architecture.

## Goals

- Keep `POST /api/v1/customer-qa/chat` stable for current clients.
- Let LLM handle intent understanding, query rewriting, and grounded answer composition.
- Keep strong fact queries on existing backend tools for products, inventory, location, and promotions.
- Add RAG for FAQ, store policy, and product knowledge that is not a real-time structured fact.
- Preserve traceability across decisioning, retrieval, tool calls, and final answer generation.
- Keep a deterministic fallback path when LLM or retrieval fails.

## Non-goals

- Building a separate agent microservice in this phase.
- Replacing all tool decisions with free-form model reasoning.
- Using a vector database in v1.
- Answering medical, legal, or financial advice questions beyond store knowledge.

## Current State

The current flow is implemented inside `backend/application/customerqa/service.go`.

- `routeIntent()` uses keyword rules.
- `answerChat()` dispatches to fixed handlers by intent.
- Product, inventory, promotion, and FAQ answers are generated with templates after repository lookups.
- Tool calls are logged, but LLM decisioning and retrieval are not modeled yet.

This gives a working backbone that can be upgraded instead of replaced.

## Recommended Architecture

Use a single backend service with an orchestrator and replaceable adapters.

```text
HTTP Handler
  -> customerqa.Service.Chat
    -> Orchestrator
      -> IntentAnalyzer (LLM)
      -> RetrievalRouter
      -> Tool Executor
      -> RAG Retriever
      -> AnswerComposer (LLM)
    -> Message and audit persistence
```

### Layering

- `api/http`
  - Keeps request validation and response mapping only.
- `application/customerqa`
  - Owns `Service.Chat`, orchestrator flow, fallback rules, and response assembly.
- `domain/customerqa`
  - Keeps repository contracts for sessions, messages, tool calls, and knowledge retrieval contracts.
- `infra/ai`
  - Implements the concrete LLM client.
- `infra/retrieval`
  - Implements FAQ and knowledge retrieval from MySQL-backed knowledge tables.

## Orchestrator Responsibilities

The orchestrator is the new control point for chat handling.

### Step 1: Intent Analysis

Input:

- User message
- Recent session messages
- Store context

Output:

```json
{
  "intent": "inventory",
  "rewritten_query": "可口可乐库存",
  "route": "tool",
  "needs_handoff": false,
  "confidence": 0.91,
  "reasoning_tags": ["structured_fact", "product_inventory"]
}
```

The model must return structured data, not free-form reasoning. The orchestrator validates and normalizes this output before execution.

### Step 2: Retrieval Routing

The orchestrator decides execution path using model output plus deterministic guards:

- `tool`
  - Product location, inventory, active promotions
- `rag`
  - FAQ, store policy, product description, non-real-time guidance
- `hybrid`
  - Questions that combine factual lookup with knowledge explanation

Hard rules override the model:

- Inventory and product location default to `tool`.
- FAQ and store-policy questions default to `rag`.
- Mixed product + rule questions use `hybrid`.
- Low confidence uses the conservative path or handoff.

### Step 3: Evidence Collection

- `tool`
  - Reuse `SearchProducts`, `GetProductLocation`, `GetInventory`, `ListActivePromotions`
- `rag`
  - Retrieve top knowledge chunks by query and intent-filtered knowledge type
- `hybrid`
  - Collect structured facts first, then retrieve supporting knowledge

All evidence is normalized into one internal structure before answer generation.

### Step 4: Answer Composition

The answer model receives:

- Original user question
- Rewritten query
- Session context
- Tool outputs
- Retrieved knowledge chunks
- Style and safety constraints

Requirements:

- Answer only from provided evidence.
- Use cautious wording for inventory, such as `系统显示`.
- If evidence is insufficient, say so explicitly.
- Refuse or downgrade high-risk advice outside store knowledge.
- Suggest human handoff when confidence or evidence is inadequate.

### Step 5: Fallback

Fallback is mandatory:

- LLM failure or timeout
  - Use the current rule-based `routeIntent()` and `answerChat()` path.
- Empty retrieval on `rag`
  - Return a conservative “no reliable knowledge found” answer.
- Tool lookup failure
  - Return current safe fallback responses and optional handoff.

## RAG Knowledge Design

### Knowledge Types

- `faq`
  - Existing store FAQ entries
- `store_policy`
  - Payment, refund, customer service, membership, abnormal-case rules
- `product_knowledge`
  - Product descriptions, aliases, tags, flavor, usage notes, caution text

### Chunking

- FAQ
  - One FAQ row equals one chunk
- Store policy
  - Chunk by topic, each chunk roughly 150-300 Chinese characters
- Product knowledge
  - Chunk by product, containing name, aliases, category, tags, description, and caution text

### Required Metadata

```json
{
  "doc_id": "faq_123",
  "store_id": 1,
  "knowledge_type": "faq",
  "title": "支付方式",
  "content": "支持微信和支付宝扫码支付",
  "tags": ["payment", "checkout"],
  "product_id": null,
  "updated_at": "2026-06-02T10:00:00Z"
}
```

### Retrieval Strategy

V1 should stay MySQL-based and lightweight:

- Keyword matching on title, content, aliases, tags
- Optional `FULLTEXT` indexing if needed
- Intent-based filtering before ranking
- Top 3-5 chunks passed to answer composition

The system should prepare for embeddings later, but not depend on vector infrastructure in this phase.

## API and Response Shape

External response compatibility is preserved:

- `session_id`
- `message_id`
- `intent`
- `answer`
- `cards`
- `handoff_required`

Extend `meta` with optional debug fields:

```json
{
  "request_id": "r1",
  "route": "hybrid",
  "confidence": 0.91,
  "rewrite_query": "可口可乐库存",
  "evidence_count": 3,
  "fallback_used": false
}
```

Default client responses should stay compact. Rich debug payloads should be gated behind internal or admin usage.

## Observability

Current tool call logging is not enough. Add dedicated records for:

- `chat_decision_log`
  - intent, route, rewritten query, confidence, fallback reason
- `retrieval_log`
  - knowledge type, retrieval query, chunk id, rank, score, selected flag
- `tool_call`
  - keep existing table, add step ordering and source if needed
- `answer_log`
  - model, prompt version, token usage, grounded flag, safety flags

This makes it possible to diagnose whether a bad answer came from intent detection, retrieval quality, tool failure, or final generation.

## Data and Interface Changes

### New Application Interfaces

- `Orchestrator`
- `IntentAnalyzer`
- `AnswerComposer`
- `Retriever`

### Domain-Level Additions

- Knowledge retrieval repository contract
- Optional persistence contract for decision and retrieval logs

### Suggested Files

- `backend/application/customerqa/orchestrator.go`
- `backend/application/customerqa/llm.go`
- `backend/application/customerqa/retriever.go`
- `backend/application/customerqa/fallback.go`
- `backend/infra/ai/openai_client.go`
- `backend/infra/retrieval/mysql_retriever.go`

## Safety Rules

- Real-time facts must come from tools, not model memory.
- RAG content must be store-scoped where relevant.
- Product guidance must not turn into medical claims.
- No chain-of-thought should be returned to the client.
- Handoff must remain available for unsupported or risky questions.

## Testing Strategy

- Orchestrator unit tests with fake LLM, fake retriever, and fake tools
- Service tests for message persistence, response mapping, and fallback
- Retrieval tests for FAQ, store policy, and product knowledge ranking
- Handler tests to preserve API compatibility

## Rollout Plan

### Phase 1

Refactor structure first:

- Introduce orchestrator interfaces
- Keep current behavior behind fallback implementation
- Add tests around routing and persistence

### Phase 2

Add real LLM integration and first RAG source:

- Structured intent analysis
- FAQ and store policy retrieval
- Evidence-based answer generation

### Phase 3

Improve hybrid routing and observability:

- Product knowledge retrieval
- Decision and retrieval logs
- Better prompt versioning and evaluation hooks

## Decision

Implement the `single backend service + orchestrator + replaceable adapters` approach.

This keeps the current deployment model, reuses the existing tool-backed business logic, and creates a controlled path to LLM + RAG without losing auditability or rollback safety.

## Implemented Limitations

The current codebase now matches the orchestration structure above, with these explicit v1 limitations:

- `backend/internal/bootstrap.Build()` still defaults to fallback-safe service wiring in runtime.
  - Primary orchestrator wiring exists and is test-covered, but production bootstrap does not yet auto-enable analyzer/composer/retriever dependencies.
- `backend/infra/ai` currently provides deterministic fake analyzer and composer adapters for tests.
  - A real OpenAI-compatible client is still pending.
- `backend/infra/retrieval/mysql_retriever.go` is FAQ-first.
  - `store_policy` is approximated from FAQ categories such as `payment`, `refund`, `store_hours`, and `customer_service`.
  - Dedicated `product_knowledge` persistence and retrieval are not implemented yet.
- Observability is partial.
  - Decision logging is supported as an optional repository capability.
  - Retrieval logs and answer logs described in this design are not persisted yet.
