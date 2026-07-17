#!/bin/bash
#
# Delegates 4.9% of total bonded stake to a single validator every 12 hours.
# Uses the reporter module queries to respect the chain's 5% stake-change window:
#   - allowed-amount: how much can be staked right now
#   - allowed-amount-expiration: when the current window resets
#
# Usage:
#   export VALIDATOR_ADDRESS="tellorvaloper..."
#   export KEY_NAME="my-key"
#   ./scripts/periodic-delegate.sh          # run forever
#   ./scripts/periodic-delegate.sh --once # single delegation cycle

set -euo pipefail

# --- configuration (override via env) ---
CHAIN_ID="${CHAIN_ID:-tellor-1}"
API_URL="${API_URL:-https://mainnet.tellorlayer.com}"
NODE_URL="${NODE_URL:-https://mainnet.tellorlayer.com/rpc}"
LAYERD_PATH="${LAYERD_PATH:-./layerd}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
KEYRING_DIR="${KEYRING_DIR:-}"
DENOM="${DENOM:-loya}"
FEES="${FEES:-20loya}"
LOG_FILE="${LOG_FILE:-/var/log/periodic-delegate.log}"
STAKE_BPS="${STAKE_BPS:-49}" # 49/1000 = 4.9% of total bonded power
GAS="${GAS:-500000}"
VALIDATOR_ADDRESS="${VALIDATOR_ADDRESS:-}"
KEY_NAME="${KEY_NAME:-}"

RUN_ONCE=false
if [[ "${1:-}" == "--once" ]]; then
    RUN_ONCE=true
fi

log_message() {
    local ts
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$ts] $1" | tee -a "$LOG_FILE"
}

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        log_message "ERROR: required command not found: $1"
        exit 1
    fi
}

query_json() {
    curl -sf "$1"
}

get_expiration_ms() {
    query_json "$API_URL/tellor-io/layer/reporter/allowed-amount-expiration" | jq -r '.expiration'
}

get_staking_allowed() {
    query_json "$API_URL/tellor-io/layer/reporter/allowed-amount" | jq -r '.staking_amount'
}

get_bonded_tokens() {
    query_json "$API_URL/cosmos/staking/v1beta1/pool" | jq -r '.pool.bonded_tokens'
}

get_available_balance() {
    local addr=$1
    local keyring_args=()
    if [[ -n "$KEYRING_DIR" ]]; then
        keyring_args=(--keyring-dir "$KEYRING_DIR")
    fi
    "$LAYERD_PATH" query bank balances "$addr" \
        --chain-id "$CHAIN_ID" \
        --node "$NODE_URL" \
        -o json \
        "${keyring_args[@]}" \
        | jq -r --arg denom "$DENOM" '.balances[] | select(.denom == $denom) | .amount // "0"'
}

get_delegator_address() {
    local keyring_args=()
    if [[ -n "$KEYRING_DIR" ]]; then
        keyring_args=(--keyring-dir "$KEYRING_DIR")
    fi
    "$LAYERD_PATH" keys show "$KEY_NAME" -a \
        --keyring-backend "$KEYRING_BACKEND" \
        "${keyring_args[@]}"
}

min_int() {
    local a=$1 b=$2
    if [[ "$a" -lt "$b" ]]; then echo "$a"; else echo "$b"; fi
}

format_time() {
    local sec=$1
    if date -r "$sec" '+%Y-%m-%d %H:%M:%S' >/dev/null 2>&1; then
        date -r "$sec" '+%Y-%m-%d %H:%M:%S'
    else
        date -d "@$sec" '+%Y-%m-%d %H:%M:%S'
    fi
}

wait_until_expiration() {
    local expiration_ms now_ms wait_ms wait_sec expiration_sec
    expiration_ms=$(get_expiration_ms)
    now_ms=$(($(date +%s) * 1000))

    if [[ "$expiration_ms" == "null" || -z "$expiration_ms" ]]; then
        log_message "WARNING: could not read expiration; proceeding immediately"
        return
    fi

    expiration_sec=$((expiration_ms / 1000))
    wait_ms=$((expiration_ms - now_ms))
    if [[ "$wait_ms" -le 0 ]]; then
        log_message "Stake window is open (expiration was $(format_time "$expiration_sec"))"
        return
    fi

    wait_sec=$((wait_ms / 1000 + 5))
    log_message "Waiting ${wait_sec}s until stake window opens at $(format_time "$expiration_sec")"
    sleep "$wait_sec"
}

execute_delegate_with_retry() {
    local amount=$1
    local max_attempts=5
    local attempt=1
    local keyring_args=()

    if [[ -n "$KEYRING_DIR" ]]; then
        keyring_args=(--keyring-dir "$KEYRING_DIR")
    fi

    while [[ $attempt -le $max_attempts ]]; do
        log_message "Delegate attempt $attempt of $max_attempts for ${amount}${DENOM}"

        local output
        if output=$("$LAYERD_PATH" tx staking delegate "$VALIDATOR_ADDRESS" "${amount}${DENOM}" \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --node "$NODE_URL" \
            --keyring-backend "$KEYRING_BACKEND" \
            --fees "$FEES" \
            --gas "$GAS" \
            --yes \
            "${keyring_args[@]}" 2>&1); then
            log_message "Delegation successful"
            log_message "$output"
            return 0
        fi

        log_message "Delegation failed: $output"
        if [[ $attempt -lt $max_attempts ]]; then
            sleep 5
        fi
        attempt=$((attempt + 1))
    done

    log_message "ERROR: delegation failed after $max_attempts attempts"
    return 1
}

run_delegation_cycle() {
    local bonded target allowed balance delegate_amount delegator_addr

    bonded=$(get_bonded_tokens)
    allowed=$(get_staking_allowed)
    delegator_addr=$(get_delegator_address)
    balance=$(get_available_balance "$delegator_addr")

    target=$((bonded * STAKE_BPS / 1000))
    delegate_amount=$(min_int "$target" "$allowed")
    delegate_amount=$(min_int "$delegate_amount" "$balance")

    log_message "Bonded tokens: $bonded"
    log_message "Target (${STAKE_BPS}/1000 of bonded): $target"
    log_message "Allowed staking amount: $allowed"
    log_message "Delegator balance: $balance"
    log_message "Delegate amount: $delegate_amount"

    if [[ -z "$delegate_amount" || "$delegate_amount" -le 0 ]]; then
        log_message "Nothing to delegate this cycle; skipping"
        return 0
    fi

    execute_delegate_with_retry "$delegate_amount"
}

# --- main ---
require_cmd curl
require_cmd jq

if [[ -z "$VALIDATOR_ADDRESS" || -z "$KEY_NAME" ]]; then
    echo "Set VALIDATOR_ADDRESS and KEY_NAME before running."
    echo "Optional: CHAIN_ID, API_URL, NODE_URL, KEYRING_BACKEND, KEYRING_DIR, LAYERD_PATH, FEES, STAKE_BPS"
    exit 1
fi

log_message "Starting periodic-delegate (validator=$VALIDATOR_ADDRESS, key=$KEY_NAME, chain=$CHAIN_ID)"

if [[ "$RUN_ONCE" == true ]]; then
    wait_until_expiration
    run_delegation_cycle
    log_message "Single cycle complete"
    exit 0
fi

while true; do
    wait_until_expiration
    run_delegation_cycle || log_message "Cycle failed; will retry at next window"
done
