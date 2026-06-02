# Store Mind Backend

Clean architecture scaffold for Store Mind backend.

## Layers

- `api`: HTTP adapters
- `application`: use-case orchestration
- `domain`: core entities and contracts
- `infra`: config, logging, persistence implementations

## Run

```bash
GO_BIN=/opt/homebrew/opt/go@1.24/bin/go
$GO_BIN mod tidy
$GO_BIN run ./cmd/server
```

## Logging

Logger uses `zap` and supports environment-based output:

- Dev mode: logs to console
- Prod mode: logs to daily files

### Environment variables

- `APP_ENV`: `production` / `prod` for production mode; any other value uses dev mode
- `LOG_LEVEL`: `debug|info|warn|error` (default `info`)
- `LOG_DIR`: output directory in production mode (default `logs`)

### Dev example

```bash
cd backend
APP_ENV=dev LOG_LEVEL=debug /opt/homebrew/opt/go@1.24/bin/go run ./cmd/server
```

### Prod example

```bash
cd backend
APP_ENV=production LOG_LEVEL=info LOG_DIR=logs /opt/homebrew/opt/go@1.24/bin/go run ./cmd/server
```

In production mode, file naming format is:

- `app-YYYY-MM-DD.log`

## Customer QA API

- `POST /api/v1/customer-qa/chat`
- `GET /api/v1/customer-qa/faqs/search?store_id=1&q=付款`
- `GET /api/v1/customer-qa/products/search?store_id=1&q=可乐`
- `GET /api/v1/customer-qa/products/{product_id}/location?store_id=1`
- `GET /api/v1/customer-qa/skus/{sku_id}/inventory?store_id=1`
- `GET /api/v1/customer-qa/promotions/active?store_id=1`
