#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

APP_PORT="${APP_PORT:-18080}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3307}"
DB_USER="${DB_USER:-app}"
DB_PASSWORD="${DB_PASSWORD:-app}"
DB_NAME="${DB_NAME:-store_mind}"
GO_BIN="${GO_BIN:-/opt/homebrew/opt/go@1.24/bin/go}"

export MYSQL_DSN="${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}?parseTime=true"
export HTTP_ADDR=":${APP_PORT}"

cleanup() {
  if [ -n "${APP_PID:-}" ] && kill -0 "$APP_PID" >/dev/null 2>&1; then
    kill "$APP_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "[smoke] start mysql"
MYSQL_PORT="$DB_PORT" docker compose up -d mysql >/dev/null

echo "[smoke] wait mysql"
for _ in {1..40}; do
  if mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "[smoke] migrate + seed"
WITH_SEED=1 DB_HOST="$DB_HOST" DB_PORT="$DB_PORT" DB_USER="$DB_USER" DB_PASSWORD="$DB_PASSWORD" DB_NAME="$DB_NAME" ./scripts/migrate.sh

echo "[smoke] run app"
"$GO_BIN" run ./cmd/server >/tmp/store-mind-smoke.log 2>&1 &
APP_PID=$!

for _ in {1..40}; do
  if curl -fsS "http://127.0.0.1:${APP_PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "[smoke] check health"
HEALTH="$(curl -fsS "http://127.0.0.1:${APP_PORT}/healthz")"
echo "$HEALTH" | rg '"status":"ok"' >/dev/null

echo "[smoke] check faq"
FAQ="$(curl -fsS "http://127.0.0.1:${APP_PORT}/api/v1/customer-qa/faqs/search?store_id=1&q=付款")"
echo "$FAQ" | rg '怎么付款|支付|付款' >/dev/null

echo "[smoke] check chat location"
CHAT_LOCATION="$(curl -fsS "http://127.0.0.1:${APP_PORT}/api/v1/customer-qa/chat" \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐在哪里"}')"
echo "$CHAT_LOCATION" | rg '"intent":"product_location"' >/dev/null
echo "$CHAT_LOCATION" | rg '饮料区|B-02' >/dev/null

echo "[smoke] check chat faq"
CHAT_FAQ="$(curl -fsS "http://127.0.0.1:${APP_PORT}/api/v1/customer-qa/chat" \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"怎么付款"}')"
echo "$CHAT_FAQ" | rg '"intent":"faq"' >/dev/null
echo "$CHAT_FAQ" | rg '微信|支付宝|扫码结算' >/dev/null

echo "SMOKE PASS"
