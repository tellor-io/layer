#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/verify-loop.bash quick [--run REGEX] PACKAGE...
  scripts/verify-loop.bash full

quick runs uncached, short, module-readonly Go tests for the named packages,
then git diff --check. full runs a readonly build, make lint, uncached short
tests for ./..., then git diff --check. E2E remains separate and explicit:
e2e/run-all-sequential.sh.

Each run writes exact commands, HEAD, initial/final porcelain-v2 status, and
per-command exit codes under private .pi/verification/.
EOF
}

if [[ ${1:-} == "--help" || ${1:-} == "-h" ]]; then
  [[ $# -eq 1 ]] || { echo "verify-loop: --help does not accept other arguments" >&2; exit 2; }
  usage
  exit 0
fi

mode=${1:-}
[[ -n "$mode" ]] || { usage >&2; exit 2; }
shift

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

runtime_dir="$repo_root/.pi"
verification_dir="$runtime_dir/verification"
mkdir -p "$verification_dir"
chmod 700 "$runtime_dir" "$verification_dir"
run_id=$(date -u +%Y%m%dT%H%M%SZ)-$$
log_file="$verification_dir/$run_id.log"
: > "$log_file"
chmod 600 "$log_file"

log_line() {
  printf '%s\n' "$*" | tee -a "$log_file"
}

log_command() {
  local rendered
  printf -v rendered '%q ' "$@"
  log_line "command: ${rendered% }"
}

run_logged() {
  log_command "$@"
  set +e
  "$@" 2>&1 | tee -a "$log_file"
  local command_status=${PIPESTATUS[0]}
  set -e
  log_line "exit_code: $command_status"
  return "$command_status"
}

finalize() {
  local run_status=$?
  set +e
  log_line "final_status_begin"
  GIT_OPTIONAL_LOCKS=0 git status --porcelain=v2 --branch | tee -a "$log_file"
  local status_code=${PIPESTATUS[0]}
  log_line "final_status_end"
  log_line "status_command_exit_code: $status_code"
  log_line "verifier_exit_code: $run_status"
  exit "$run_status"
}
trap finalize EXIT

log_line "run_id: $run_id"
log_line "mode: $mode"
log_line "head: $(git rev-parse HEAD)"
log_line "initial_status_begin"
GIT_OPTIONAL_LOCKS=0 git status --porcelain=v2 --branch | tee -a "$log_file"
initial_status_code=${PIPESTATUS[0]}
log_line "initial_status_end"
log_line "status_command_exit_code: $initial_status_code"
[[ $initial_status_code -eq 0 ]] || exit "$initial_status_code"

case "$mode" in
  quick)
    run_regex=""
    if [[ ${1:-} == "--run" ]]; then
      [[ $# -ge 3 ]] || { echo "verify-loop: quick --run requires REGEX and at least one PACKAGE" >&2; exit 2; }
      run_regex=$2
      shift 2
    fi
    [[ $# -gt 0 ]] || { echo "verify-loop: quick requires at least one PACKAGE" >&2; exit 2; }
    for package in "$@"; do
      [[ "$package" != -* ]] || { echo "verify-loop: unexpected option or package: $package" >&2; exit 2; }
    done
    go_test=(go test -mod=readonly -short -count=1)
    [[ -z "$run_regex" ]] || go_test+=(-run "$run_regex")
    go_test+=("$@")
    run_logged "${go_test[@]}"
    run_logged git diff --check
    ;;
  full)
    [[ $# -eq 0 ]] || { echo "verify-loop: full does not accept arguments" >&2; exit 2; }
    run_logged go build -mod=readonly ./...
    run_logged make lint
    run_logged go test -mod=readonly -short -count=1 ./...
    run_logged git diff --check
    ;;
  *)
    echo "verify-loop: expected quick or full" >&2
    usage >&2
    exit 2
    ;;
esac
