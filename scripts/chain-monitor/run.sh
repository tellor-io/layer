#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="${1:-$ROOT/example-config.yml}"
shift || true

cd "$ROOT"
exec go run ./cmd/chain-monitor -config="$CONFIG" "$@"
