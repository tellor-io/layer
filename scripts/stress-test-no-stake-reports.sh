#!/bin/bash

# Stress test script for no stake report transactions
# Sends multiple no stake reports with random query data just under the size limit

# Don't exit on error - we want to continue even if some transactions fail
set +e

# Configuration
# Path to layerd binary (default: "layerd" - assumes it's in PATH, or set to full path like "./layerd")
LAYERD_PATH="${LAYERD_PATH:-layerd}"
NODE="http://localhost:26657"
CHAIN_ID="layer"
KEYRING_BACKEND="test"
KEY_NAME="validator"
FEE_AMOUNT="500loya"
VALUE="100"  # Simple value for the report

# Size limit is 524288 bytes (0.5MB), use 524000 to be safely under
# Number of transactions to send (can be overridden with first argument)
NUM_TXS=${1:-10}

# Delay between transactions in seconds (can be overridden with second argument)
DELAY=${2:-1}

# Query data size in bytes (can be overridden with third argument, default: 524000)
QUERY_DATA_SIZE=${3:-524000}

# Timeout duration for unordered transactions (default: 1 minute)
TIMEOUT_DURATION="${TIMEOUT_DURATION:-1m}"

echo "=== No Stake Report Stress Test ==="
echo "Layerd path: $LAYERD_PATH"
echo "Node: $NODE"
echo "Chain ID: $CHAIN_ID"
echo "Number of transactions: $NUM_TXS"
echo "Query data size: $QUERY_DATA_SIZE bytes"
echo "Delay between transactions: $DELAY seconds"
echo "Timeout duration: $TIMEOUT_DURATION"
echo ""
echo "Usage: $0 [num_txs] [delay] [query_data_size]"
echo "  num_txs: Number of transactions to send (default: 10)"
echo "  delay: Delay between transactions in seconds (default: 1)"
echo "  query_data_size: Size of query data in bytes (default: 524000)"
echo ""
echo "Using unordered transactions (--unordered flag) - no sequence management needed!"
echo ""
echo "Note: Set LAYERD_PATH environment variable to specify layerd binary path"
echo "  Example: export LAYERD_PATH=./layerd"
echo ""

# Check if required commands are available
# Check if layerd exists (either in PATH or as a file)
if ! command -v "$LAYERD_PATH" &> /dev/null; then
    # If not in PATH, check if it's a file (for relative/absolute paths)
    if [ ! -f "$LAYERD_PATH" ]; then
        echo "Error: layerd not found at '$LAYERD_PATH'"
        echo "Please set LAYERD_PATH environment variable or ensure layerd is in your PATH."
        echo "Example: export LAYERD_PATH=./layerd"
        exit 1
    fi
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq command not found. Please install jq (e.g., 'brew install jq' or 'apt-get install jq')."
    exit 1
fi

if ! command -v openssl &> /dev/null; then
    echo "Error: openssl command not found. Please ensure openssl is installed."
    exit 1
fi

# Check if node is accessible
if ! curl -s "$NODE/status" > /dev/null 2>&1; then
    echo "Error: Cannot connect to node at $NODE"
    echo "Please ensure a node is running locally on default ports."
    exit 1
fi

echo "Node is accessible. Starting stress test..."
echo ""

# Function to generate random hex query data
generate_random_query_data() {
    # Generate random bytes and convert to hex
    # openssl rand -hex N generates N random bytes and outputs as 2N hex characters
    # We want QUERY_DATA_SIZE bytes, so we generate QUERY_DATA_SIZE bytes
    openssl rand -hex $QUERY_DATA_SIZE | tr -d '\n'
}

# Send transactions
SUCCESS_COUNT=0
FAIL_COUNT=0

for i in $(seq 1 $NUM_TXS); do
    echo "[$i/$NUM_TXS] Generating random query data and sending transaction..."
    
    # Generate random query data
    query_data=$(generate_random_query_data)
    
    # Send the transaction with unordered flag (no sequence management needed)
    result=$("$LAYERD_PATH" tx oracle no-stake-report "$query_data" "$VALUE" \
        --chain-id "$CHAIN_ID" \
        --from "$KEY_NAME" \
        --fees "$FEE_AMOUNT" \
        --keyring-backend "$KEYRING_BACKEND" \
        --node "$NODE" \
        --unordered \
        --timeout-duration "$TIMEOUT_DURATION" \
        --yes \
        --output json 2>&1) || true
    
    # Check if transaction was successful
    txhash=$(echo "$result" | jq -r '.txhash' 2>/dev/null || echo "")
    code=$(echo "$result" | jq -r '.code' 2>/dev/null || echo "")
    
    if [ -n "$txhash" ] && ([ "$code" = "0" ] || [ "$code" = "null" ] || [ -z "$code" ]); then
        echo "  ✓ Success: TxHash=$txhash"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        # Extract error message if available
        error_msg=$(echo "$result" | jq -r '.raw_log' 2>/dev/null || echo "$result" | tail -1)
        echo "  ✗ Failed: $error_msg"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
    
    # Wait before next transaction (except after the last one)
    if [ $i -lt $NUM_TXS ]; then
        sleep "$DELAY"
    fi
done

echo ""
echo "=== Stress Test Complete ==="
echo "Successful transactions: $SUCCESS_COUNT"
echo "Failed transactions: $FAIL_COUNT"
echo "Total transactions: $NUM_TXS"

