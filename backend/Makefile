GO_BIN ?= /opt/homebrew/opt/go@1.24/bin/go

.PHONY: tidy test run migrate-up seed smoke

tidy:
	$(GO_BIN) mod tidy

test:
	$(GO_BIN) test ./...

run:
	$(GO_BIN) run ./cmd/server

migrate-up:
	DB_HOST=127.0.0.1 DB_PORT=3307 DB_USER=app DB_PASSWORD=app DB_NAME=store_mind ./scripts/migrate.sh

seed:
	WITH_SEED=1 DB_HOST=127.0.0.1 DB_PORT=3307 DB_USER=app DB_PASSWORD=app DB_NAME=store_mind ./scripts/migrate.sh

smoke:
	bash scripts/smoke.sh
