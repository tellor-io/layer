#!/usr/bin/env bash
# Run each e2e test individually and print a summary report.
#
# Usage (from repo root or e2e/):
#   ./e2e/run-all-sequential.sh
#   TIMEOUT=20m ./e2e/run-all-sequential.sh
#   E2E_RACE=1 ./e2e/run-all-sequential.sh
#
# Options via environment:
#   TIMEOUT           Per-test timeout (default: 15m)
#   E2E_RACE          Set to 1 to pass -race to go test (slower)
#   E2E_DOCKER_SWEEP  Set to "" to skip the daemon-wide ibc-test docker sweep
#                     TestMain runs around each test process (default: 1)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

TIMEOUT="${TIMEOUT:-15m}"
# The sweep removes every docker object with interchaintest's ibc-test label,
# from any repo sharing the daemon — see main_test.go. Serial suite runs want
# it; ad-hoc `go test -run` invocations leave it off unless exported.
export E2E_DOCKER_SWEEP="${E2E_DOCKER_SWEEP-1}"
RACE_FLAG=()
if [[ "${E2E_RACE:-}" == "1" ]]; then
  RACE_FLAG=(-race)
fi

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not on PATH" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is not on PATH" >&2
  exit 1
fi
if ! docker image inspect layer:local >/dev/null 2>&1; then
  echo "error: layer:local docker image not found (is the docker daemon running?); run 'make local-image' first" >&2
  exit 1
fi

# Discover tests with go test -list (same as CI prepare).
tests=()
list_out="$(go test -list '^Test' .)" || {
  echo "error: go test -list failed in ${SCRIPT_DIR}" >&2
  exit 1
}
while IFS= read -r name; do
  [[ -n "$name" ]] || continue
  tests+=("$name")
done < <(printf '%s\n' "$list_out" | grep '^Test' || true)

if [[ ${#tests[@]} -eq 0 ]]; then
  echo "error: no tests found in ${SCRIPT_DIR}" >&2
  exit 1
fi

# Colors (disabled when not a tty).
if [[ -t 1 ]]; then
  GREEN=$'\033[32m'
  RED=$'\033[31m'
  DIM=$'\033[2m'
  BOLD=$'\033[1m'
  RESET=$'\033[0m'
else
  GREEN="" RED="" DIM="" BOLD="" RESET=""
fi

format_duration() {
  local secs=$1
  if (( secs >= 3600 )); then
    printf '%dh %dm %ds' $((secs / 3600)) $(((secs % 3600) / 60)) $((secs % 60))
  elif (( secs >= 60 )); then
    printf '%dm %ds' $((secs / 60)) $((secs % 60))
  else
    printf '%ds' "$secs"
  fi
}

started_at=$(date '+%Y-%m-%d %H:%M:%S')
suite_start=$SECONDS

passed=0
failed=0

declare -a results_name=()
declare -a results_status=()
declare -a results_secs=()
declare -a failed_names=()

echo "${BOLD}Running ${#tests[@]} e2e tests sequentially${RESET} ${DIM}(timeout=${TIMEOUT} each)${RESET}"
echo

for i in "${!tests[@]}"; do
  test_name="${tests[$i]}"
  n=$((i + 1))
  total=${#tests[@]}

  echo "${BOLD}[${n}/${total}]${RESET} ${test_name}"
  test_start=$SECONDS

  go test -v -count=1 ${RACE_FLAG[@]+"${RACE_FLAG[@]}"} -run "^${test_name}\$" -timeout "$TIMEOUT" . 2>&1 | sed "s/^/  /"
  exit_code=${PIPESTATUS[0]}

  test_secs=$((SECONDS - test_start))
  results_name+=("$test_name")
  results_secs+=("$test_secs")

  if [[ $exit_code -eq 0 ]]; then
    passed=$((passed + 1))
    results_status+=("PASS")
    echo "  ${GREEN}PASS${RESET} ${DIM}($(format_duration "$test_secs"))${RESET}"
  else
    failed=$((failed + 1))
    results_status+=("FAIL")
    failed_names+=("$test_name")
    echo "  ${RED}FAIL${RESET} ${DIM}($(format_duration "$test_secs"), exit ${exit_code})${RESET}"
  fi

  # Build-cache growth causes the same startup faults ~1.5h in; trim periodically.
  if (( n % 10 == 0 )) && command -v docker >/dev/null 2>&1; then
    docker builder prune -f --filter until=2h >/dev/null 2>&1 || true
  fi
  echo
done

suite_secs=$((SECONDS - suite_start))
width=52

echo "${BOLD}================================================================${RESET}"
printf "${BOLD}  E2E Test Report${RESET}  ${DIM}%s${RESET}\n" "$started_at"
echo "${BOLD}================================================================${RESET}"
echo

for i in "${!results_name[@]}"; do
  name="${results_name[$i]}"
  status="${results_status[$i]}"
  secs="${results_secs[$i]}"
  dur="$(format_duration "$secs")"

  if [[ "$status" == "PASS" ]]; then
    mark="${GREEN}PASS${RESET}"
  else
    mark="${RED}FAIL${RESET}"
  fi

  if ((${#name} > width)); then
    printf "  %b  %s\n       %s\n" "$mark" "${name:0:width}…" "$dur"
  else
    printf "  %b  %-${width}s  %s\n" "$mark" "$name" "$dur"
  fi
done

echo
echo "${BOLD}----------------------------------------------------------------${RESET}"
printf "  Total:   %d\n" "${#results_name[@]}"
printf "  ${GREEN}Passed:${RESET}  %d\n" "$passed"
printf "  ${RED}Failed:${RESET}  %d\n" "$failed"
printf "  Elapsed: %s\n" "$(format_duration "$suite_secs")"
echo "${BOLD}----------------------------------------------------------------${RESET}"

if [[ ${#failed_names[@]} -gt 0 ]]; then
  echo
  echo "${RED}${BOLD}Failed tests:${RESET}"
  for name in "${failed_names[@]}"; do
    echo "  - $name"
  done
  echo
  echo "${DIM}Re-run one test:${RESET}"
  echo "  cd e2e && go test -v -count=1 -run '^${failed_names[0]}\$' -timeout ${TIMEOUT}"
  exit 1
fi

exit 0
