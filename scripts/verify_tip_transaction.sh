#!/bin/bash

# Script to sign, broadcast, and verify tip transactions
# Usage: ./verify_tip_transaction.sh [key_name] [chain_id] [node_url] [tx_file]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        print_error "$1 is not installed. Please install it first."
        exit 1
    fi
}

# Function to check if file exists
check_file() {
    if [ ! -f "$1" ]; then
        print_error "File $1 does not exist."
        exit 1
    fi
}

# Function to extract value from JSON using jq
extract_json_value() {
    local json="$1"
    local key="$2"
    echo "$json" | jq -r "$key" 2>/dev/null || echo ""
}

# Function to wait for user confirmation
wait_for_confirmation() {
    echo
    read -p "Press Enter to continue or Ctrl+C to abort..."
}

# Check required commands
check_command "jq"
check_command "layerd"

# Parse command line arguments
if [ $# -lt 4 ]; then
    echo "Usage: $0 <key_name> <chain_id> <node_url> <tx_file>"
    echo "Example: $0 mykey layer-testnet http://localhost:26657 tx.json"
    exit 1
fi

KEY_NAME="$1"
CHAIN_ID="$2"
NODE_URL="$3"
TX_FILE="$4"

# Validate inputs
check_file "$TX_FILE"

print_status "Starting tip transaction verification..."
print_status "Key: $KEY_NAME"
print_status "Chain ID: $CHAIN_ID"
print_status "Node URL: $NODE_URL"
print_status "Transaction file: $TX_FILE"

# Check if key exists
if ! layerd keys show "$KEY_NAME" &>/dev/null; then
    print_error "Key '$KEY_NAME' not found. Please check your keyring."
    exit 1
fi

# Create temporary files
SIGNED_TX_FILE=$(mktemp)
TX_RESULT_FILE=$(mktemp)
AGGREGATE_RESULT_FILE=$(mktemp)

# Cleanup function
cleanup() {
    rm -f "$SIGNED_TX_FILE" "$TX_RESULT_FILE" "$AGGREGATE_RESULT_FILE"
}

# Set trap to cleanup on exit
trap cleanup EXIT

print_status "Step 1: Signing transaction..."
if ! layerd tx sign "$TX_FILE" \
    --from "$KEY_NAME" \
    --chain-id "$CHAIN_ID" \
    --node "$NODE_URL" \
    --output json \
    --yes > "$SIGNED_TX_FILE" 2>&1; then
    print_error "Failed to sign transaction"
    cat "$SIGNED_TX_FILE"
    exit 1
fi

print_success "Transaction signed successfully"

print_status "Step 2: Broadcasting transaction..."
BROADCAST_OUTPUT=$(layerd tx broadcast "$SIGNED_TX_FILE" \
    --node "$NODE_URL" \
    --output json \
    --yes 2>&1)

if echo "$BROADCAST_OUTPUT" | jq -e '.txhash' >/dev/null 2>&1; then
    TX_HASH=$(echo "$BROADCAST_OUTPUT" | jq -r '.txhash')
    print_success "Transaction broadcasted successfully"
    print_status "Transaction hash: $TX_HASH"
else
    print_error "Failed to broadcast transaction"
    echo "$BROADCAST_OUTPUT"
    exit 1
fi

print_status "Step 3: Querying transaction result..."
# Wait a moment for transaction to be included in a block
sleep 3

# Query transaction with retries
MAX_RETRIES=10
RETRY_COUNT=0
TX_RESULT=""

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if TX_RESULT=$(layerd query tx "$TX_HASH" \
        --node "$NODE_URL" \
        --output json 2>/dev/null); then
        
        # Check if transaction was successful
        TX_CODE=$(echo "$TX_RESULT" | jq -r '.tx_result.code // empty')
        if [ "$TX_CODE" = "0" ] || [ -z "$TX_CODE" ]; then
            print_success "Transaction result retrieved successfully"
            break
        else
            print_error "Transaction failed with code: $TX_CODE"
            echo "$TX_RESULT" | jq -r '.tx_result.log // empty'
            exit 1
        fi
    else
        RETRY_COUNT=$((RETRY_COUNT + 1))
        print_warning "Transaction not found yet, retrying... ($RETRY_COUNT/$MAX_RETRIES)"
        sleep 2
    fi
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    print_error "Failed to retrieve transaction result after $MAX_RETRIES attempts"
    exit 1
fi

print_status "Step 4: Extracting tip events..."
# Extract all tip_added events with their query_id and querymeta_id
TIP_EVENTS=$(echo "$TX_RESULT" | jq -r '
.tx_result.events[] | 
select(.type == "tip_added") | 
{
    query_id: (.attributes[] | select(.key == "query_id") | .value),
    querymeta_id: (.attributes[] | select(.key == "querymeta_id") | .value),
    tipper: (.attributes[] | select(.key == "tipper") | .value),
    amount: (.attributes[] | select(.key == "amount") | .value)
}' 2>/dev/null)

if [ -z "$TIP_EVENTS" ] || [ "$TIP_EVENTS" = "null" ]; then
    print_error "No tip_added events found in transaction"
    exit 1
fi

TIP_COUNT=$(echo "$TIP_EVENTS" | jq -r 'length')
print_success "Found $TIP_COUNT tip events"

print_status "Step 5: Waiting for blockchain processing..."
sleep 5

print_status "Step 6: Verifying aggregate reports..."
VERIFICATION_RESULTS=""
VERIFICATION_COUNT=0
SUCCESS_COUNT=0
FAILURE_COUNT=0

# Process each tip event
echo "$TIP_EVENTS" | jq -c '.[]' | while read -r event; do
    VERIFICATION_COUNT=$((VERIFICATION_COUNT + 1))
    
    QUERY_ID=$(echo "$event" | jq -r '.query_id')
    QUERYMETA_ID=$(echo "$event" | jq -r '.querymeta_id')
    TIPPER=$(echo "$event" | jq -r '.tipper')
    AMOUNT=$(echo "$event" | jq -r '.amount')
    
    print_status "Verifying tip $VERIFICATION_COUNT/$TIP_COUNT"
    print_status "  Query ID: $QUERY_ID"
    print_status "  Expected Meta ID: $QUERYMETA_ID"
    print_status "  Tipper: $TIPPER"
    print_status "  Amount: $AMOUNT"
    
    # Query aggregate report
    if AGGREGATE_RESULT=$(layerd query oracle get-current-aggregate-report "$QUERY_ID" \
        --node "$NODE_URL" \
        --output json 2>/dev/null); then
        
        AGGREGATE_META_ID=$(echo "$AGGREGATE_RESULT" | jq -r '.aggregate.meta_id // empty')
        
        if [ -n "$AGGREGATE_META_ID" ] && [ "$AGGREGATE_META_ID" != "null" ]; then
            if [ "$AGGREGATE_META_ID" = "$QUERYMETA_ID" ]; then
                print_success "  ✓ Meta ID matches: $AGGREGATE_META_ID"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
                print_error "  ✗ Meta ID mismatch: expected $QUERYMETA_ID, got $AGGREGATE_META_ID"
                FAILURE_COUNT=$((FAILURE_COUNT + 1))
            fi
        else
            print_warning "  ⚠ No aggregate report found for query ID: $QUERY_ID"
            FAILURE_COUNT=$((FAILURE_COUNT + 1))
        fi
    else
        print_error "  ✗ Failed to query aggregate report for query ID: $QUERY_ID"
        FAILURE_COUNT=$((FAILURE_COUNT + 1))
    fi
    
    echo
done

# Wait for the while loop to complete
wait

print_status "Step 7: Summary"
print_status "Transaction hash: $TX_HASH"
print_status "Total tips verified: $TIP_COUNT"
print_status "Successful verifications: $SUCCESS_COUNT"
print_status "Failed verifications: $FAILURE_COUNT"

if [ $FAILURE_COUNT -eq 0 ]; then
    print_success "All tip verifications passed! ✓"
    exit 0
else
    print_error "Some tip verifications failed! ✗"
    exit 1
fi
