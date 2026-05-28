# Customer QA Agent Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an MVP customer-facing QA Agent for an unmanned supermarket that answers product location, inventory, promotion, and store FAQ questions using structured store data and auditable tool calls.

**Architecture:** Use a small monorepo with a business API service, an Agent service, and shared database migrations. The business API owns store/product/inventory/promotion/FAQ data; the Agent service owns intent routing, tool calling, answer generation, and conversation logs. Keep the MVP lightweight and avoid Kafka, Temporal, vector DB, and IoT dependencies until the customer QA loop is stable.

**Tech Stack:** Go API service, Python Agent service, MySQL, optional Redis cache, OpenAI-compatible LLM API, HTTP JSON APIs, Docker Compose for local development.

---

## Task 1: Create Project Skeleton

**Files:**
- Create: `README.md`
- Create: `docker-compose.yml`
- Create: `services/api/README.md`
- Create: `services/agent/README.md`
- Create: `db/migrations/README.md`

**Step 1: Write the project README**

Create `README.md` with:

```markdown
# Unmanned Supermarket Customer QA Agent

MVP for a customer-facing Agent that answers store questions using structured product, shelf, inventory, promotion, and FAQ data.

## Services

- `services/api`: business data API for store, product, inventory, promotion, FAQ, and logs
- `services/agent`: customer QA Agent, intent routing, tool calling, and answer generation
- `db/migrations`: MySQL schema migrations

## MVP Scope

- Product search
- Product location answers
- Inventory answers
- Promotion answers
- Store FAQ answers
- Agent session, message, and tool-call logs
```

**Step 2: Add local infrastructure**

Create `docker-compose.yml`:

```yaml
services:
  mysql:
    image: mysql:8.4
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: unmanned_supermarket
      MYSQL_USER: app
      MYSQL_PASSWORD: app
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql

volumes:
  mysql-data:
```

**Step 3: Add service placeholders**

Create short README files in `services/api`, `services/agent`, and `db/migrations` explaining the responsibility of each directory.

**Step 4: Verify files exist**

Run:

```bash
find . -maxdepth 3 -type f | sort
```

Expected: all skeleton files are listed.

**Step 5: Commit**

```bash
git add README.md docker-compose.yml services db
git commit -m "chore: create customer qa agent project skeleton"
```

## Task 2: Add Database Schema

**Files:**
- Create: `db/migrations/001_customer_qa_agent_schema.sql`
- Create: `db/seeds/001_demo_store_data.sql`

**Step 1: Write schema migration**

Create tables:

- `store`
- `zone`
- `shelf`
- `product`
- `sku`
- `product_location`
- `inventory`
- `promotion`
- `faq`
- `agent_session`
- `agent_message`
- `agent_tool_call`

Use the fields defined in `docs/plans/2026-05-28-customer-qa-agent-design.md`.

**Step 2: Add indexes**

Add indexes for:

- `zone.store_id`
- `shelf.store_id`
- `shelf.zone_id`
- `product.name`
- `sku.product_id`
- `inventory.store_id, inventory.sku_id`
- `promotion.store_id, promotion.status, promotion.start_at, promotion.end_at`
- `faq.store_id, faq.status`
- `agent_message.session_id`
- `agent_tool_call.session_id`

**Step 3: Add seed data**

Create demo data with:

- One store
- Three zones: drinks, snacks, daily goods
- Six shelves
- Ten products
- Ten SKUs
- Inventory for each SKU
- Three promotions
- Ten FAQ entries

**Step 4: Apply migration locally**

Run:

```bash
docker compose up -d mysql
mysql -h127.0.0.1 -uapp -papp unmanned_supermarket < db/migrations/001_customer_qa_agent_schema.sql
mysql -h127.0.0.1 -uapp -papp unmanned_supermarket < db/seeds/001_demo_store_data.sql
```

Expected: both commands complete without SQL errors.

**Step 5: Verify schema**

Run:

```bash
mysql -h127.0.0.1 -uapp -papp -e "USE unmanned_supermarket; SHOW TABLES;"
```

Expected: all twelve MVP tables are present.

**Step 6: Commit**

```bash
git add db/migrations/001_customer_qa_agent_schema.sql db/seeds/001_demo_store_data.sql
git commit -m "feat: add customer qa agent database schema"
```

## Task 3: Implement Business API Foundations

**Files:**
- Create: `services/api/go.mod`
- Create: `services/api/cmd/api/main.go`
- Create: `services/api/internal/config/config.go`
- Create: `services/api/internal/db/db.go`
- Create: `services/api/internal/http/router.go`
- Create: `services/api/internal/http/health_handler.go`
- Create: `services/api/internal/http/health_handler_test.go`

**Step 1: Write failing health test**

Create a test that calls `GET /healthz` and expects:

```json
{"status":"ok"}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd services/api && go test ./...
```

Expected: fail because router and handler do not exist yet.

**Step 3: Implement minimal API server**

Implement:

- Config loading from environment.
- MySQL connection helper.
- HTTP router.
- `GET /healthz`.
- Main server entrypoint on `:8080`.

**Step 4: Run tests**

Run:

```bash
cd services/api && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api
git commit -m "feat: add api service foundation"
```

## Task 4: Implement Product, Location, Inventory, Promotion, and FAQ APIs

**Files:**
- Create: `services/api/internal/catalog/repository.go`
- Create: `services/api/internal/catalog/repository_test.go`
- Create: `services/api/internal/catalog/handlers.go`
- Create: `services/api/internal/catalog/handlers_test.go`
- Modify: `services/api/internal/http/router.go`

**Step 1: Write repository tests**

Add tests for:

- Search product by exact name.
- Search product by alias.
- Search product by category.
- Get product location.
- Get SKU inventory.
- List active promotions.
- Search FAQ by keyword.

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/api && go test ./internal/catalog -v
```

Expected: fail because repository functions do not exist.

**Step 3: Implement repository**

Implement:

- `SearchProducts(ctx, storeID, query string)`
- `GetProductLocation(ctx, storeID, productID int64)`
- `GetInventory(ctx, storeID, skuID int64)`
- `ListActivePromotions(ctx, storeID int64, now time.Time)`
- `SearchFAQ(ctx, storeID int64, query string)`

**Step 4: Add HTTP handlers**

Expose:

- `GET /api/products/search?store_id=1&q=可乐`
- `GET /api/products/{product_id}/location?store_id=1`
- `GET /api/skus/{sku_id}/inventory?store_id=1`
- `GET /api/promotions/active?store_id=1`
- `GET /api/faqs/search?store_id=1&q=怎么付款`

**Step 5: Run tests**

Run:

```bash
cd services/api && go test ./...
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api/internal/catalog services/api/internal/http/router.go
git commit -m "feat: add catalog query APIs"
```

## Task 5: Implement Agent Service Foundation

**Files:**
- Create: `services/agent/pyproject.toml`
- Create: `services/agent/app/main.py`
- Create: `services/agent/app/config.py`
- Create: `services/agent/app/models.py`
- Create: `services/agent/app/api_client.py`
- Create: `services/agent/tests/test_health.py`

**Step 1: Write failing health test**

Test `GET /healthz` returns:

```json
{"status":"ok"}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd services/agent && pytest
```

Expected: fail because app does not exist.

**Step 3: Implement FastAPI app**

Implement:

- Config loading.
- Health route.
- Business API client wrapper.

**Step 4: Run tests**

Run:

```bash
cd services/agent && pytest
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/agent
git commit -m "feat: add agent service foundation"
```

## Task 6: Implement Agent Intent Router

**Files:**
- Create: `services/agent/app/intent.py`
- Create: `services/agent/tests/test_intent.py`

**Step 1: Write intent tests**

Cover:

- “可乐在哪里” -> `product_location`
- “可乐还有吗” -> `inventory`
- “今天有什么优惠” -> `promotion`
- “怎么付款” -> `faq`
- “我要找人工” -> `handoff`
- “帮我写论文” -> `unsupported`

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/agent && pytest tests/test_intent.py -v
```

Expected: fail because intent router does not exist.

**Step 3: Implement rule-first router**

Implement a deterministic router using keyword patterns first. Leave an interface for LLM-based fallback later.

**Step 4: Run tests**

Run:

```bash
cd services/agent && pytest tests/test_intent.py -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/agent/app/intent.py services/agent/tests/test_intent.py
git commit -m "feat: add agent intent router"
```

## Task 7: Implement Agent Tools

**Files:**
- Create: `services/agent/app/tools.py`
- Create: `services/agent/tests/test_tools.py`

**Step 1: Write tool tests with mocked API client**

Cover:

- `search_products`
- `get_product_location`
- `get_inventory`
- `search_promotions`
- `search_faq`
- API failure returns structured error

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/agent && pytest tests/test_tools.py -v
```

Expected: fail because tools do not exist.

**Step 3: Implement tools**

Each tool returns:

```json
{
  "success": true,
  "data": {},
  "error": null
}
```

or:

```json
{
  "success": false,
  "data": null,
  "error": "message"
}
```

**Step 4: Run tests**

Run:

```bash
cd services/agent && pytest tests/test_tools.py -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/agent/app/tools.py services/agent/tests/test_tools.py
git commit -m "feat: add agent data tools"
```

## Task 8: Implement Answer Generation

**Files:**
- Create: `services/agent/app/answers.py`
- Create: `services/agent/tests/test_answers.py`

**Step 1: Write answer tests**

Cover:

- Product location answer includes zone, shelf, and position.
- Inventory answer uses “系统显示”.
- Promotion answer includes activity title and validity.
- FAQ answer uses approved FAQ text.
- Unsupported answer redirects to store-related questions.
- Missing data answer does not fabricate facts.

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/agent && pytest tests/test_answers.py -v
```

Expected: fail because answer generator does not exist.

**Step 3: Implement template-first answers**

Use deterministic templates for MVP. Add an LLM polish hook only after tests pass, and keep templates as fallback.

**Step 4: Run tests**

Run:

```bash
cd services/agent && pytest tests/test_answers.py -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/agent/app/answers.py services/agent/tests/test_answers.py
git commit -m "feat: add safe answer generation"
```

## Task 9: Implement Chat Endpoint and Logging

**Files:**
- Modify: `services/agent/app/main.py`
- Create: `services/agent/app/chat.py`
- Create: `services/agent/tests/test_chat.py`
- Create: `services/api/internal/agentlogs/repository.go`
- Create: `services/api/internal/agentlogs/handlers.go`
- Create: `services/api/internal/agentlogs/handlers_test.go`
- Modify: `services/api/internal/http/router.go`

**Step 1: Write chat tests**

Cover:

- `POST /api/agent/chat` returns answer, intent, session ID, and cards.
- Tool calls are logged.
- Unsupported question returns safe refusal.
- API tool failure returns fallback answer.

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/agent && pytest tests/test_chat.py -v
cd services/api && go test ./internal/agentlogs -v
```

Expected: fail because chat and log APIs do not exist.

**Step 3: Implement API log endpoints**

Expose internal endpoints for Agent service:

- `POST /api/internal/agent/sessions`
- `POST /api/internal/agent/messages`
- `POST /api/internal/agent/tool-calls`
- `GET /api/admin/agent/sessions`
- `GET /api/admin/agent/tool-calls`

**Step 4: Implement Agent chat orchestration**

Flow:

1. Create or reuse session.
2. Save user message.
3. Route intent.
4. Call required tool.
5. Generate answer.
6. Save assistant message.
7. Save tool call.
8. Return response.

**Step 5: Run tests**

Run:

```bash
cd services/api && go test ./...
cd services/agent && pytest
```

Expected: PASS.

**Step 6: Commit**

```bash
git add services/api services/agent
git commit -m "feat: add customer chat endpoint and agent logs"
```

## Task 10: Add End-to-End Smoke Test

**Files:**
- Create: `scripts/smoke_customer_qa.sh`
- Create: `docs/testing/customer-qa-smoke.md`

**Step 1: Write smoke script**

The script should:

- Start MySQL.
- Apply migrations.
- Apply seed data.
- Start API service.
- Start Agent service.
- Ask “可乐在哪里？”
- Assert response contains “饮料区”.
- Ask “怎么付款？”
- Assert response contains payment FAQ content.

**Step 2: Run smoke test**

Run:

```bash
bash scripts/smoke_customer_qa.sh
```

Expected: exits with code 0 and prints `SMOKE PASS`.

**Step 3: Document manual QA**

Add manual test cases:

- Product location.
- Inventory.
- Promotion.
- FAQ.
- Unsupported question.
- Missing product.
- Tool failure fallback.

**Step 4: Commit**

```bash
git add scripts/smoke_customer_qa.sh docs/testing/customer-qa-smoke.md
git commit -m "test: add customer qa smoke coverage"
```

## Task 11: Add MVP Admin Maintenance APIs

**Files:**
- Create: `services/api/internal/admin/handlers.go`
- Create: `services/api/internal/admin/handlers_test.go`
- Modify: `services/api/internal/http/router.go`

**Step 1: Write admin API tests**

Cover create/update/list for:

- Products
- SKUs
- Inventory
- Promotions
- FAQ

**Step 2: Run tests to verify failure**

Run:

```bash
cd services/api && go test ./internal/admin -v
```

Expected: fail because admin handlers do not exist.

**Step 3: Implement minimal CRUD**

Implement only fields required by the Agent MVP. Add validation:

- Product name required.
- SKU product ID required.
- Inventory quantity cannot be negative.
- Promotion start time must be before end time.
- FAQ question and answer required.

**Step 4: Run tests**

Run:

```bash
cd services/api && go test ./...
```

Expected: PASS.

**Step 5: Commit**

```bash
git add services/api/internal/admin services/api/internal/http/router.go
git commit -m "feat: add mvp admin maintenance APIs"
```

## Task 12: Final Verification

**Files:**
- Modify: `README.md`

**Step 1: Update README run instructions**

Document:

- How to start MySQL.
- How to apply migrations and seeds.
- How to run API tests.
- How to run Agent tests.
- How to run smoke test.
- Example chat request.

**Step 2: Run all tests**

Run:

```bash
cd services/api && go test ./...
cd services/agent && pytest
bash scripts/smoke_customer_qa.sh
```

Expected:

- Go tests PASS.
- Python tests PASS.
- Smoke test prints `SMOKE PASS`.

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add customer qa agent runbook"
```

## Open Decisions Before Implementation

- Confirm whether the first user interface is miniapp, web chat, or store screen.
- Confirm whether MVP requires voice input, or if text-only is acceptable for Phase 1.
- Confirm preferred LLM provider and deployment constraints.
- Confirm whether the backend must be Go, or whether Node.js is acceptable.
- Confirm whether MySQL is already available in the deployment environment.

## Recommended Execution

Start with Tasks 1-4 to build reliable store data access before touching LLM logic. Then implement Tasks 5-9 to add the Agent loop. Keep answer generation template-first until the tool and logging path is stable; use LLM polishing only after deterministic behavior is covered by tests.
