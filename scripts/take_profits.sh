#!/usr/bin/env bash

set -uo pipefail

usage() {
  cat <<'USAGE'
Usage:
  ./take_profits.sh <chain_id> <account_name>

Environment overrides:
  LAYERD=./layerd          layerd binary to use
  NODE=http://...          RPC endpoint
  FEES=10loya             transaction fee
  GAS=auto                gas setting
  KEYRING_BACKEND=test    optional keyring backend
  KEYRING_DIR=/path       optional keyring directory
  HOME_DIR=/path          optional layerd home directory

The script queries profit state first, then asks before every transaction.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

CHAIN_ID="$1"
ACCOUNT_NAME="$2"
FEES="${FEES:-10loya}"
GAS="${GAS:-auto}"
ACTIONS=()

if [[ -n "${LAYERD:-}" ]]; then
  LAYERD_BIN="$LAYERD"
elif [[ -x "./layerd" ]]; then
  LAYERD_BIN="./layerd"
else
  LAYERD_BIN="$(command -v layerd 2>/dev/null || true)"
fi

if [[ -z "$LAYERD_BIN" ]]; then
  echo "Could not find layerd. Set LAYERD=/path/to/layerd and try again." >&2
  exit 1
fi

NODE="${NODE:-}"
KEYRING_BACKEND="${KEYRING_BACKEND:-}"
KEYRING_DIR="${KEYRING_DIR:-}"
HOME_DIR="${HOME_DIR:-}"

prompt_default() {
  local prompt="$1"
  local default="$2"
  local value

  if [[ -n "$default" ]]; then
    read -r -p "$prompt [$default]: " value
    printf '%s' "${value:-$default}"
  else
    read -r -p "$prompt: " value
    printf '%s' "$value"
  fi
}

confirm() {
  local prompt="$1"
  local answer

  read -r -p "$prompt [y/N]: " answer
  case "$answer" in
    y|Y|yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}

append_if_set() {
  local -n target="$1"
  local flag="$2"
  local value="$3"

  if [[ -n "$value" ]]; then
    target+=("$flag" "$value")
  fi
}

build_query_flags() {
  QUERY_FLAGS=()
  append_if_set QUERY_FLAGS "--node" "$NODE"
  append_if_set QUERY_FLAGS "--keyring-backend" "$KEYRING_BACKEND"
  append_if_set QUERY_FLAGS "--keyring-dir" "$KEYRING_DIR"
  append_if_set QUERY_FLAGS "--home" "$HOME_DIR"
}

build_tx_flags() {
  TX_FLAGS=("--from" "$ACCOUNT_NAME" "--chain-id" "$CHAIN_ID" "--fees" "$FEES" "--gas" "$GAS" "--yes")
  append_if_set TX_FLAGS "--node" "$NODE"
  append_if_set TX_FLAGS "--keyring-backend" "$KEYRING_BACKEND"
  append_if_set TX_FLAGS "--keyring-dir" "$KEYRING_DIR"
  append_if_set TX_FLAGS "--home" "$HOME_DIR"
}

build_key_flags() {
  KEY_FLAGS=()
  append_if_set KEY_FLAGS "--keyring-backend" "$KEYRING_BACKEND"
  append_if_set KEY_FLAGS "--keyring-dir" "$KEYRING_DIR"
  append_if_set KEY_FLAGS "--home" "$HOME_DIR"
}

run_query() {
  echo
  echo "+ $LAYERD_BIN $*"
  if ! "$LAYERD_BIN" "$@"; then
    echo "Query failed; continuing because this state may not exist for this account yet." >&2
  fi
}

run_tx() {
  echo
  echo "+ $LAYERD_BIN $*"
  "$LAYERD_BIN" "$@"
}

validate_amount() {
  local amount="$1"
  [[ "$amount" =~ ^[0-9]+loya$ ]]
}

normalize_eth_address() {
  local address="$1"
  address="${address#0x}"
  address="${address#0X}"
  printf '%s' "$address"
}

validate_eth_address() {
  local address="$1"
  [[ "$address" =~ ^[0-9a-fA-F]{40}$ ]]
}

sum_loya_json() {
  if ! command -v python3 >/dev/null 2>&1; then
    echo "n/a"
    return
  fi

  python3 -c '
import json
import sys
from decimal import Decimal

def walk(value):
    if isinstance(value, dict):
        if value.get("denom") == "loya" and "amount" in value:
            try:
                yield Decimal(str(value["amount"]))
            except Exception:
                pass
        for child in value.values():
            yield from walk(child)
    elif isinstance(value, list):
        for child in value:
            yield from walk(child)

try:
    data = json.load(sys.stdin)
    total = sum(walk(data), Decimal(0))
    print(format(total.normalize(), "f"))
except Exception:
    print("n/a")
'
}

extract_json_field() {
  local field="$1"

  if ! command -v python3 >/dev/null 2>&1; then
    echo "n/a"
    return
  fi

  FIELD="$field" python3 -c '
import json
import os
import sys
from decimal import Decimal

field = os.environ["FIELD"]

def find(value):
    if isinstance(value, dict):
        if field in value:
            return value[field]
        for child in value.values():
            found = find(child)
            if found is not None:
                return found
    elif isinstance(value, list):
        for child in value:
            found = find(child)
            if found is not None:
                return found
    return None

try:
    data = json.load(sys.stdin)
    found = find(data)
    if found is None:
        print("n/a")
    else:
        print(format(Decimal(str(found)).normalize(), "f"))
except Exception:
    print("n/a")
'
}

metric_sum_loya() {
  local output

  if ! output="$("$LAYERD_BIN" "$@" "${QUERY_FLAGS[@]}" --output json 2>/dev/null)"; then
    echo "n/a"
    return
  fi
  printf '%s' "$output" | sum_loya_json
}

metric_field() {
  local field="$1"
  shift
  local output

  if ! output="$("$LAYERD_BIN" "$@" "${QUERY_FLAGS[@]}" --output json 2>/dev/null)"; then
    echo "n/a"
    return
  fi
  printf '%s' "$output" | extract_json_field "$field"
}

decimal_delta() {
  local before="$1"
  local after="$2"

  if [[ "$before" == "n/a" || "$after" == "n/a" ]]; then
    echo "n/a"
    return
  fi

  if ! command -v python3 >/dev/null 2>&1; then
    echo "n/a"
    return
  fi

  BEFORE="$before" AFTER="$after" python3 -c '
import os
from decimal import Decimal

try:
    delta = Decimal(os.environ["AFTER"]) - Decimal(os.environ["BEFORE"])
    print(format(delta.normalize(), "f"))
except Exception:
    print("n/a")
'
}

record_action() {
  local status="$1"
  local description="$2"
  ACTIONS+=("$status|$description")
}

print_action_report() {
  echo
  echo "=== Action Summary ==="
  printf '%-10s | %s\n' "Status" "Action"
  printf '%-10s-+-%s\n' "----------" "------------------------------------------------------------"

  if [[ ${#ACTIONS[@]} -eq 0 ]]; then
    printf '%-10s | %s\n' "NONE" "No actions recorded"
    return
  fi

  local entry status description
  for entry in "${ACTIONS[@]}"; do
    status="${entry%%|*}"
    description="${entry#*|}"
    printf '%-10s | %s\n' "$status" "$description"
  done
}

print_metric_report() {
  local final_bank="$1"
  local final_rewards="$2"
  local final_commission="$3"
  local final_tips="$4"
  local final_reporter_stake="$5"
  local final_signer_stake="$6"
  local final_unbonding="$7"

  echo
  echo "=== Profit Numbers ==="
  echo "Amounts are in loya. Positive delta means the number went up."
  printf '%-34s | %18s | %18s | %18s\n' "Metric" "Before" "After" "Delta"
  printf '%-34s-+-%18s-+-%18s-+-%18s\n' "----------------------------------" "------------------" "------------------" "------------------"
  printf '%-34s | %18s | %18s | %18s\n' "Liquid bank balance" "$BEFORE_BANK" "$final_bank" "$(decimal_delta "$BEFORE_BANK" "$final_bank")"
  printf '%-34s | %18s | %18s | %18s\n' "Validator rewards claimable" "$BEFORE_VALIDATOR_REWARDS" "$final_rewards" "$(decimal_delta "$BEFORE_VALIDATOR_REWARDS" "$final_rewards")"
  printf '%-34s | %18s | %18s | %18s\n' "Validator commission claimable" "$BEFORE_VALIDATOR_COMMISSION" "$final_commission" "$(decimal_delta "$BEFORE_VALIDATOR_COMMISSION" "$final_commission")"
  printf '%-34s | %18s | %18s | %18s\n' "Reporter available-tips" "$BEFORE_REPORTER_TIPS" "$final_tips" "$(decimal_delta "$BEFORE_REPORTER_TIPS" "$final_tips")"
  printf '%-34s | %18s | %18s | %18s\n' "Reporter reward delegation" "$BEFORE_REPORTER_STAKE" "$final_reporter_stake" "$(decimal_delta "$BEFORE_REPORTER_STAKE" "$final_reporter_stake")"
  printf '%-34s | %18s | %18s | %18s\n' "Signer active delegation" "$BEFORE_SIGNER_STAKE" "$final_signer_stake" "$(decimal_delta "$BEFORE_SIGNER_STAKE" "$final_signer_stake")"
  printf '%-34s | %18s | %18s | %18s\n' "Signer unbonding balance" "$BEFORE_UNBONDING" "$final_unbonding" "$(decimal_delta "$BEFORE_UNBONDING" "$final_unbonding")"
}

require_value() {
  local name="$1"
  local value="$2"

  if [[ -z "$value" ]]; then
    echo "$name is required." >&2
    exit 1
  fi
}

echo "Tellor Operator Profit Taker"
echo "Chain ID: $CHAIN_ID"
echo "Account key: $ACCOUNT_NAME"
echo "Binary: $LAYERD_BIN"

NODE="$(prompt_default "RPC node endpoint, leave empty to use local/default config" "$NODE")"
KEYRING_BACKEND="$(prompt_default "Keyring backend, leave empty to use layerd default" "$KEYRING_BACKEND")"
KEYRING_DIR="$(prompt_default "Keyring directory, leave empty to use layerd default" "$KEYRING_DIR")"
HOME_DIR="$(prompt_default "layerd home directory, leave empty to use layerd default" "$HOME_DIR")"
FEES="$(prompt_default "Fee for each transaction" "$FEES")"
GAS="$(prompt_default "Gas setting for each transaction" "$GAS")"

build_query_flags
build_tx_flags
build_key_flags

ACCOUNT_ADDRESS="$("$LAYERD_BIN" keys show "$ACCOUNT_NAME" -a "${KEY_FLAGS[@]}")"
if [[ -z "$ACCOUNT_ADDRESS" ]]; then
  echo "Could not resolve account address for key '$ACCOUNT_NAME'." >&2
  exit 1
fi

DERIVED_VALIDATOR="$("$LAYERD_BIN" keys show "$ACCOUNT_NAME" --bech val -a "${KEY_FLAGS[@]}" 2>/dev/null || true)"
VALIDATOR_ADDRESS="$(prompt_default "Validator operator address for rewards/staking" "$DERIVED_VALIDATOR")"
SELECTOR_ADDRESS="$(prompt_default "Selector address for reporter rewards" "$ACCOUNT_ADDRESS")"
REPORTER_REWARD_VALIDATOR="$(prompt_default "Bonded validator to receive withdrawn reporter rewards" "$VALIDATOR_ADDRESS")"

require_value "Validator operator address" "$VALIDATOR_ADDRESS"
require_value "Selector address" "$SELECTOR_ADDRESS"
require_value "Bonded validator for reporter rewards" "$REPORTER_REWARD_VALIDATOR"
REPORT_SIGNER_VALIDATOR="$REPORTER_REWARD_VALIDATOR"

echo
echo "Resolved settings:"
echo "  account address: $ACCOUNT_ADDRESS"
echo "  validator address: $VALIDATOR_ADDRESS"
echo "  selector address: $SELECTOR_ADDRESS"
echo "  reporter rewards stake to: $REPORTER_REWARD_VALIDATOR"
echo "  fees: $FEES"
echo "  gas: $GAS"

if [[ "$SELECTOR_ADDRESS" != "$ACCOUNT_ADDRESS" ]]; then
  echo
  echo "Note: reporter rewards can be checked for $SELECTOR_ADDRESS, but staking unbond"
  echo "transactions below use the signer key '$ACCOUNT_NAME' ($ACCOUNT_ADDRESS)."
  echo "To unbond selector stake, run this script with the selector's key as account_name."
fi

echo
echo "=== Before: current profit state ==="
BEFORE_BANK="$(metric_sum_loya query bank balances "$ACCOUNT_ADDRESS")"
BEFORE_VALIDATOR_REWARDS="$(metric_sum_loya query distribution rewards-by-validator "$ACCOUNT_ADDRESS" "$VALIDATOR_ADDRESS")"
BEFORE_VALIDATOR_COMMISSION="$(metric_sum_loya query distribution commission "$VALIDATOR_ADDRESS")"
BEFORE_REPORTER_TIPS="$(metric_field "available_tips" query reporter available-tips "$SELECTOR_ADDRESS")"
BEFORE_REPORTER_STAKE="$(metric_sum_loya query staking delegation "$SELECTOR_ADDRESS" "$REPORTER_REWARD_VALIDATOR")"
BEFORE_SIGNER_STAKE="$(metric_sum_loya query staking delegation "$ACCOUNT_ADDRESS" "$REPORTER_REWARD_VALIDATOR")"
BEFORE_UNBONDING="$(metric_sum_loya query staking unbonding-delegation "$ACCOUNT_ADDRESS" "$REPORTER_REWARD_VALIDATOR")"
run_query query bank balances "$ACCOUNT_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query distribution rewards-by-validator "$ACCOUNT_ADDRESS" "$VALIDATOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query distribution commission "$VALIDATOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query oracle get-time-based-rewards "${QUERY_FLAGS[@]}"
run_query query reporter selector-reporter "$SELECTOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query reporter available-tips "$SELECTOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query staking delegation "$SELECTOR_ADDRESS" "$REPORTER_REWARD_VALIDATOR" "${QUERY_FLAGS[@]}"

echo
echo "=== Claim validator profits ==="
if confirm "Withdraw delegation rewards from $VALIDATOR_ADDRESS?"; then
  if run_tx tx distribution withdraw-rewards "$VALIDATOR_ADDRESS" "${TX_FLAGS[@]}"; then
    record_action "SUCCESS" "Withdrew delegation rewards from $VALIDATOR_ADDRESS"
  else
    record_action "FAILED" "Withdrew delegation rewards from $VALIDATOR_ADDRESS"
  fi
else
  record_action "SKIPPED" "Delegation reward withdrawal"
  echo "Skipped delegation reward withdrawal."
fi

if confirm "Withdraw validator commission from $VALIDATOR_ADDRESS?"; then
  if run_tx tx distribution withdraw-validator-commission "$VALIDATOR_ADDRESS" "${TX_FLAGS[@]}"; then
    record_action "SUCCESS" "Withdrew validator commission from $VALIDATOR_ADDRESS"
  else
    record_action "FAILED" "Withdrew validator commission from $VALIDATOR_ADDRESS"
  fi
else
  record_action "SKIPPED" "Validator commission withdrawal"
  echo "Skipped validator commission withdrawal."
fi

echo
echo "=== Claim reporter profits ==="
echo "Reporter rewards are staked directly; they do not become liquid balance."
if confirm "Withdraw reporter available-tips for $SELECTOR_ADDRESS and stake them to $REPORTER_REWARD_VALIDATOR?"; then
  if run_tx tx reporter withdraw-tip "$SELECTOR_ADDRESS" "$REPORTER_REWARD_VALIDATOR" "${TX_FLAGS[@]}"; then
    record_action "SUCCESS" "Withdrew reporter available-tips for $SELECTOR_ADDRESS to $REPORTER_REWARD_VALIDATOR"
  else
    record_action "FAILED" "Withdrew reporter available-tips for $SELECTOR_ADDRESS to $REPORTER_REWARD_VALIDATOR"
  fi
else
  record_action "SKIPPED" "Reporter available-tips withdrawal"
  echo "Skipped reporter reward withdrawal."
fi

echo
echo "=== After claims: updated profit state ==="
run_query query bank balances "$ACCOUNT_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query distribution rewards-by-validator "$ACCOUNT_ADDRESS" "$VALIDATOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query distribution commission "$VALIDATOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query reporter available-tips "$SELECTOR_ADDRESS" "${QUERY_FLAGS[@]}"
run_query query staking delegation "$SELECTOR_ADDRESS" "$REPORTER_REWARD_VALIDATOR" "${QUERY_FLAGS[@]}"

echo
echo "=== Optional: begin unbonding ==="
run_query query staking params "${QUERY_FLAGS[@]}"
run_query query staking delegation "$ACCOUNT_ADDRESS" "$REPORTER_REWARD_VALIDATOR" "${QUERY_FLAGS[@]}"
run_query query staking unbonding-delegation "$ACCOUNT_ADDRESS" "$REPORTER_REWARD_VALIDATOR" "${QUERY_FLAGS[@]}"

if confirm "Begin unbonding staked TRB so it can become liquid later?"; then
  UNBOND_VALIDATOR="$(prompt_default "Validator to unbond from" "$REPORTER_REWARD_VALIDATOR")"
  UNBOND_AMOUNT="$(prompt_default "Amount to unbond, in loya, for example 1000000loya" "")"

  require_value "Validator to unbond from" "$UNBOND_VALIDATOR"
  REPORT_SIGNER_VALIDATOR="$UNBOND_VALIDATOR"
  BEFORE_SIGNER_STAKE="$(metric_sum_loya query staking delegation "$ACCOUNT_ADDRESS" "$REPORT_SIGNER_VALIDATOR")"
  BEFORE_UNBONDING="$(metric_sum_loya query staking unbonding-delegation "$ACCOUNT_ADDRESS" "$REPORT_SIGNER_VALIDATOR")"

  if ! validate_amount "$UNBOND_AMOUNT"; then
    echo "Invalid unbond amount '$UNBOND_AMOUNT'. Expected a whole loya coin such as 1000000loya." >&2
    record_action "FAILED" "Validated unbond amount $UNBOND_AMOUNT"
  elif confirm "Submit unbond transaction for $UNBOND_AMOUNT from $UNBOND_VALIDATOR?"; then
    if run_tx tx staking unbond "$UNBOND_VALIDATOR" "$UNBOND_AMOUNT" "${TX_FLAGS[@]}"; then
      record_action "SUCCESS" "Began unbonding $UNBOND_AMOUNT from $UNBOND_VALIDATOR"
    else
      record_action "FAILED" "Began unbonding $UNBOND_AMOUNT from $UNBOND_VALIDATOR"
    fi
    run_query query staking delegation "$ACCOUNT_ADDRESS" "$UNBOND_VALIDATOR" "${QUERY_FLAGS[@]}"
    run_query query staking unbonding-delegation "$ACCOUNT_ADDRESS" "$UNBOND_VALIDATOR" "${QUERY_FLAGS[@]}"
  else
    record_action "SKIPPED" "Unbond transaction"
    echo "Skipped unbond transaction."
  fi
else
  record_action "SKIPPED" "Unbonding"
  echo "Skipped unbonding."
fi

echo
echo "=== Optional: bridge liquid TRB to Ethereum ==="
echo "Only liquid bank balance can be bridged. Unbonding tokens must finish first."
run_query query bank balances "$ACCOUNT_ADDRESS" "${QUERY_FLAGS[@]}"

if confirm "Initiate a bridge withdrawal to Ethereum?"; then
  BRIDGE_AMOUNT="$(prompt_default "Amount to bridge, in loya, for example 1000000loya" "")"
  ETH_RECIPIENT="$(normalize_eth_address "$(prompt_default "Ethereum recipient address, with or without 0x" "")")"

  if ! validate_amount "$BRIDGE_AMOUNT"; then
    echo "Invalid bridge amount '$BRIDGE_AMOUNT'. Expected a whole loya coin such as 1000000loya." >&2
    record_action "FAILED" "Validated bridge amount $BRIDGE_AMOUNT"
  elif ! validate_eth_address "$ETH_RECIPIENT"; then
    echo "Invalid Ethereum recipient '$ETH_RECIPIENT'. Expected 40 hex characters." >&2
    record_action "FAILED" "Validated Ethereum recipient $ETH_RECIPIENT"
  elif confirm "Burn $BRIDGE_AMOUNT on Layer and withdraw to Ethereum recipient $ETH_RECIPIENT?"; then
    if run_tx tx bridge withdraw-tokens "$ACCOUNT_ADDRESS" "$ETH_RECIPIENT" "$BRIDGE_AMOUNT" "${TX_FLAGS[@]}"; then
      record_action "SUCCESS" "Initiated bridge withdrawal of $BRIDGE_AMOUNT to $ETH_RECIPIENT"
    else
      record_action "FAILED" "Initiated bridge withdrawal of $BRIDGE_AMOUNT to $ETH_RECIPIENT"
    fi
    run_query query bank balances "$ACCOUNT_ADDRESS" "${QUERY_FLAGS[@]}"
  else
    record_action "SKIPPED" "Bridge withdrawal transaction"
    echo "Skipped bridge withdrawal."
  fi
else
  record_action "SKIPPED" "Bridge withdrawal"
  echo "Skipped bridge withdrawal."
fi

FINAL_BANK="$(metric_sum_loya query bank balances "$ACCOUNT_ADDRESS")"
FINAL_VALIDATOR_REWARDS="$(metric_sum_loya query distribution rewards-by-validator "$ACCOUNT_ADDRESS" "$VALIDATOR_ADDRESS")"
FINAL_VALIDATOR_COMMISSION="$(metric_sum_loya query distribution commission "$VALIDATOR_ADDRESS")"
FINAL_REPORTER_TIPS="$(metric_field "available_tips" query reporter available-tips "$SELECTOR_ADDRESS")"
FINAL_REPORTER_STAKE="$(metric_sum_loya query staking delegation "$SELECTOR_ADDRESS" "$REPORTER_REWARD_VALIDATOR")"
FINAL_SIGNER_STAKE="$(metric_sum_loya query staking delegation "$ACCOUNT_ADDRESS" "$REPORT_SIGNER_VALIDATOR")"
FINAL_UNBONDING="$(metric_sum_loya query staking unbonding-delegation "$ACCOUNT_ADDRESS" "$REPORT_SIGNER_VALIDATOR")"

print_action_report
print_metric_report \
  "$FINAL_BANK" \
  "$FINAL_VALIDATOR_REWARDS" \
  "$FINAL_VALIDATOR_COMMISSION" \
  "$FINAL_REPORTER_TIPS" \
  "$FINAL_REPORTER_STAKE" \
  "$FINAL_SIGNER_STAKE" \
  "$FINAL_UNBONDING"

echo
echo "Done. Review the before/after query output to confirm where profits moved."
