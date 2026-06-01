#!/usr/bin/env bash
set -euo pipefail

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3307}"
DB_USER="${DB_USER:-app}"
DB_PASSWORD="${DB_PASSWORD:-app}"
DB_NAME="${DB_NAME:-store_mind}"

run_sql_dir() {
  local dir="$1"
  if [ ! -d "$dir" ]; then
    return 0
  fi

  local files
  files=$(cd "$dir" && ls -1 *.sql 2>/dev/null | sort || true)
  if [ -z "$files" ]; then
    return 0
  fi

  while IFS= read -r f; do
    [ -z "$f" ] && continue
    echo "[migrate] applying $dir/$f"
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < "$dir/$f"
  done <<< "$files"
}

run_sql_dir "db/migrations"
if [ "${WITH_SEED:-0}" = "1" ]; then
  run_sql_dir "db/seeds"
fi

echo "[migrate] done"
