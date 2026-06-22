#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PYTHON="${S1_PYTHON:-python3}"
RUNNER="${S1_RUNNER:-$ROOT/backend/scripts/s1_acceptance.py}"

usage() {
  cat <<'EOF'
Usage: bash backend/scripts/s1_verify.sh [options]

Options:
  --mode local|ci          local diagnostics or non-interactive CI gate
  --api-base URL           Go backend base URL
  --sidecar-base URL       Python sidecar base URL
  --output-dir PATH        report directory
  --help                    show this help

Environment:
  S1_API_BASE, S1_SIDECAR_BASE, S1_OUTPUT_DIR
  S1_MYSQL_HOST, S1_MYSQL_PORT, S1_MYSQL_USER
  S1_MYSQL_PASSWORD, S1_MYSQL_DATABASE
  S1_PYTHON, S1_RUNNER

(S0_* variants for each env var are also recognised as fallback.)
EOF
}

mode="local"
args=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      [[ $# -ge 2 ]] || { echo "--mode requires a value" >&2; exit 2; }
      mode="$2"
      args+=("$1" "$2")
      shift 2
      ;;
    --api-base|--sidecar-base|--output-dir)
      [[ $# -ge 2 ]] || { echo "$1 requires a value" >&2; exit 2; }
      args+=("$1" "$2")
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "$mode" != "local" && "$mode" != "ci" ]]; then
  echo "--mode must be local or ci" >&2
  exit 2
fi

if [[ ! " ${args[*]} " =~ " --mode " ]]; then
  args=("--mode" "$mode" "${args[@]}")
fi

if [[ "$mode" == "local" ]]; then
  echo "Running S1 acceptance in local mode"
else
  echo "Running S1 acceptance in CI mode"
fi

exec "$PYTHON" "$RUNNER" "${args[@]}"
