# Store Mind Backend Clean Architecture Design

Date: 2026-05-29

## Scope

Build a production-leaning backend scaffold using Go + Gin + GORM + MySQL, with a first bounded context `customer_qa`.

## Goals

- Provide a clean architecture baseline with strict dependency direction.
- Deliver one runnable business path for `customer_qa` chat session/message persistence.
- Include config, logging middleware, unified error response, DI bootstrap, DB migration skeleton, and tests.

## Non-goals

- Full feature parity with Coze Studio.
- Multi-service split, message bus, advanced authn/authz.
- Complex domain richness in v1.

## Architecture

Layers and dependencies:

- `api` -> `application` -> `domain`
- `infra` implements repository interfaces defined by `domain` / used by `application`
- `cmd/server` and `internal/bootstrap` wire dependencies

No inward layer imports outward layers.

## Directory Plan

- `cmd/server/main.go`
- `api/http/{router.go,handler_customer_qa.go,middleware.go,response.go}`
- `application/customerqa/service.go`
- `domain/customerqa/{entity.go,repository.go,errors.go}`
- `infra/config/config.go`
- `infra/logger/logger.go`
- `infra/persistence/mysql/{models.go,repository_customer_qa.go,db.go}`
- `internal/bootstrap/app.go`
- `db/migrations/001_init_customer_qa.sql`
- `tests` via package-level `_test.go`

## Data Flow

`POST /api/v1/customer-qa/chat`

1. Handler validates request.
2. Application service executes use case.
3. Service persists session/message through domain repository interface.
4. Repository implementation stores in MySQL with GORM.
5. Handler returns normalized response.

## Error Handling

- Domain and application return typed errors.
- API maps errors to stable JSON: `{code,message}`.
- Unknown errors map to `internal_error` with HTTP 500.

## Observability

- Request logging middleware with method/path/status/latency.
- Boot logs for config and DB connectivity.

## Testing

- Application unit test with in-memory fake repo.
- API handler test via `httptest` and fake app service.

## Evolution Path

- Add more contexts under `domain/*` and `application/*`.
- Optionally split into modular monolith by context.
