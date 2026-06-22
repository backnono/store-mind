#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT/backend/scripts/s0_verify.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cat > "$TMP/fake_runner.py" <<'PY'
import os
import sys
from pathlib import Path

Path(os.environ["CAPTURE_FILE"]).write_text("\n".join(sys.argv[1:]))
raise SystemExit(int(os.environ.get("FAKE_EXIT", "0")))
PY

CAPTURE_FILE="$TMP/args.txt" \
S0_RUNNER="$TMP/fake_runner.py" \
S0_PYTHON=python3 \
bash "$SCRIPT" --mode ci --skip-intent-eval --output-dir "$TMP/report"

grep -qx -- "--mode" "$TMP/args.txt"
grep -qx -- "ci" "$TMP/args.txt"
grep -qx -- "--skip-intent-eval" "$TMP/args.txt"
grep -qx -- "--output-dir" "$TMP/args.txt"
grep -qx -- "$TMP/report" "$TMP/args.txt"

set +e
CAPTURE_FILE="$TMP/args-fail.txt" \
FAKE_EXIT=7 \
S0_RUNNER="$TMP/fake_runner.py" \
S0_PYTHON=python3 \
bash "$SCRIPT" --mode local
status=$?
set -e

if [[ "$status" -ne 7 ]]; then
  echo "expected runner exit code 7, got $status" >&2
  exit 1
fi

if bash "$SCRIPT" --mode invalid >/dev/null 2>&1; then
  echo "invalid mode unexpectedly succeeded" >&2
  exit 1
fi

bash "$SCRIPT" --help | grep -q -- "--mode local|ci"
