#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PYTHON="${S0_PYTHON:-python3}"
RUNNER="${S0_RUNNER:-$ROOT/backend/scripts/s0_acceptance.py}"

usage() {
  cat <<'EOF'
Usage: bash backend/scripts/s0_verify.sh [options]

Options:
  --mode local|ci          local diagnostics or non-interactive CI gate
  --api-base URL           Go backend base URL
  --sidecar-base URL       Python sidecar base URL
  --output-dir PATH        report directory
  --skip-intent-eval       skip the 56-case real-LLM quality gate
  --help                    show this help

Environment:
  S0_API_BASE, S0_SIDECAR_BASE, S0_OUTPUT_DIR
  S0_MYSQL_HOST, S0_MYSQL_PORT, S0_MYSQL_USER
  S0_MYSQL_PASSWORD, S0_MYSQL_DATABASE
  S0_PYTHON, S0_RUNNER
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
    --skip-intent-eval)
      args+=("$1")
      shift
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
  echo "Running S0 acceptance in local mode"
else
  echo "Running S0 acceptance in CI mode"
fi

exec "$PYTHON" "$RUNNER" "${args[@]}"
