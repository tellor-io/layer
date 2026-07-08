#!/usr/bin/env bash
# Fail on new unreviewed production stakingKeeper.Delegate call sites outside
# the explicit allowlist. Complements ADR 1012 governance/keeper-delegation review.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

ALLOWLIST=(
  "x/reporter/keeper/stake_return.go"
  "x/reporter/keeper/msg_server.go"
)

declare -A allowed=()
for file in "${ALLOWLIST[@]}"; do
  allowed[$file]=1
done

mapfile -t sites < <(
  rg -n 'stakingKeeper\.Delegate\(' x/reporter x/dispute \
    --glob '*.go' \
    --glob '!*_test.go' \
    --glob '!**/mocks/**' \
    2>/dev/null || true
)

if [[ ${#sites[@]} -eq 0 ]]; then
  exit 0
fi

failed=0
for line in "${sites[@]}"; do
  file="${line%%:*}"
  if [[ -z "${allowed[$file]:-}" ]]; then
    echo "unreviewed keeper Delegate site: $line" >&2
    failed=1
  fi
done

exit "$failed"
