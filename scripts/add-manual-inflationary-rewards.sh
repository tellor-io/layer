#!/bin/bash

# Set up logging
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Function to log messages
log_message() {
    echo "[$TIMESTAMP] $1"
}

# Stop execution if any command fails
set -e

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed. Please install jq to continue."
    echo "On macOS: brew install jq"
    echo "On Ubuntu/Debian: sudo apt-get install jq"
    echo "On CentOS/RHEL: sudo yum install jq"
    exit 1
fi

# Check if layerd is available
if ! command -v ./layerd &> /dev/null; then
    echo "Error: layerd binary not found in current directory."
    echo "Please run this script from the directory containing the layerd binary."
    exit 1
fi

# Configuration variables
KEYRING_BACKEND="test"
CHAIN_ID="layertest-4"
FEES="20loya"
LAYER_HOME="$HOME/.layer"
RPC_NODE="https://node-palmito.tellorlayer.com/rpc/"

# Check if we can connect to the RPC node
log_message "Testing connection to RPC node: $RPC_NODE"
if ! ./layerd query block --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "Error: Cannot connect to RPC node: $RPC_NODE"
    echo "Please check your internet connection and the RPC node URL."
    exit 1
fi
log_message "Successfully connected to RPC node"

# Account addresses - UPDATE THESE WITH YOUR ACTUAL ADDRESSES
export TIPPER_ACCOUNT="tellor14au4s3t0z59nlkk39npcqk7d2snld5090d2pk3"  # Your tipper account
export TIME_REWARDS_POOL="tellor1364k288xd4r0gxlnk2rve5fm3qxamdtted7v4t"  # Time-based rewards pool address
export VALIDATOR_FEE_COLLECTOR="tellor17xpfvakm2amg962yls6f84z3kell8c5ls06m3g"  # Validator fee collector address

# Calculate amounts
TIME_REWARDS_AMOUNT=525
VALIDATOR_FEE_AMOUNT=175


log_message "Starting inflationary rewards script"

# Validate configuration
if [ -z "$TIPPER_ACCOUNT" ] || [ -z "$TIME_REWARDS_POOL" ] || [ -z "$VALIDATOR_FEE_COLLECTOR" ]; then
    log_message "Error: Required account addresses are not set"
    log_message "Please update the script with valid addresses"
    exit 1
fi

log_message "Configuration:"
log_message "  Chain ID: $CHAIN_ID"
log_message "  RPC Node: $RPC_NODE"
log_message "  Layer Home: $LAYER_HOME"
log_message "  Tipper Account: $TIPPER_ACCOUNT"
log_message "  Time Rewards Pool: $TIME_REWARDS_POOL"
log_message "  Validator Fee Collector: $VALIDATOR_FEE_COLLECTOR"
log_message "  Total Rewards per Block: ${TOTAL_REWARDS_PER_BLOCK}loya"
log_message "  Time Rewards: ${TIME_REWARDS_AMOUNT}loya (${TIME_REWARDS_PERCENTAGE}%)"
log_message "  Validator Fees: ${VALIDATOR_FEE_AMOUNT}loya (${VALIDATOR_FEE_PERCENTAGE}%)"

# Function to get current block height
get_current_block() {
    echo "calling get_current_block"
    local block_height
    # Get the raw output and filter to only JSON lines, then parse with jq
    local raw_output
    raw_output=$(./layerd query block --node https://node-palmito.tellorlayer.com/rpc/ --output json 2>&1)
    
    # Filter to only lines that contain JSON structure (start with { or [)
    local json_output
    json_output=$(echo "$raw_output" | grep -E '^[[:space:]]*[{\[]')
    
    # Parse the filtered JSON
    block_height=$(echo "$json_output" | jq -r '.header.height')
    echo "block_height: $block_height"
    
    # Validate the block height
    if [ -z "$block_height" ] || [ "$block_height" = "null" ] || [ "$block_height" -lt 0 ]; then
        log_message "Error: Failed to get valid block height. Raw output:"
        echo "$raw_output" | head -20
        return 1
    fi
    
    echo "$block_height"
}

# Function to wait for next block
wait_for_next_block() {
    local current_block=$1
    local next_block=$((current_block + 1))
    
    log_message "Waiting for block $next_block..."
    
    while true; do
        local current
        if ! current=$(get_current_block); then
            log_message "Warning: Failed to get current block, retrying in 5 seconds..."
            sleep 5
            continue
        fi
        
        if [ "$current" -ge "$next_block" ]; then
            log_message "Block $next_block reached"
            break
        fi
        sleep 1
    done
}

# Function to execute transaction with retry
execute_with_retry() {
    local cmd="$1"
    local description="$2"
    local max_attempts=5
    local attempt=1
    local success=false

    while [ $attempt -le $max_attempts ] && [ "$success" = false ]; do
        log_message "Attempt $attempt of $max_attempts for $description"
        
        # Capture the output of the command
        local output
        output=$(eval "$cmd" 2>&1)
        
        # Show the full transaction output
        log_message "Transaction output:"
        log_message "$output"
        log_message "----------------------------------------"
        
        # Extract raw_log field using grep
        local raw_log
        raw_log=$(echo "$output" | grep "raw_log:" | cut -d':' -f2- | sed 's/^[[:space:]]*//')
        
        # Check if raw_log is empty or contains an error message
        if [ "$raw_log" = '""' ] || [ "$raw_log" = "''" ]; then
            success=true
            log_message "Transaction successful for $description"
        else
            log_message "Transaction failed for $description, attempt $attempt of $max_attempts"
            log_message "Error: $raw_log"
            if [ $attempt -lt $max_attempts ]; then
                log_message "Waiting 5 seconds before retry..."
                sleep 5
            fi
        fi
        attempt=$((attempt + 1))
    done

    if [ "$success" = false ]; then
        log_message "Failed to execute transaction for $description after $max_attempts attempts"
        return 1
    fi
}

# Function to send rewards for a single block
send_block_rewards() {
    local block_height=$1
    
    log_message "Sending rewards for block $block_height"
    
    # Get account sequence and account number
    if ! get_account_info "$TIPPER_ACCOUNT"; then
        log_message "Error: Failed to get account info"
        return 1
    fi
    
    # Create batched transaction for this block
    create_batched_transaction "$block_height"
    
    # Sign and broadcast the batched transaction
    log_message "Signing and broadcasting batched transaction"
    local WORKING_DIR=$(pwd)
    
    ./layerd tx sign $WORKING_DIR/tx_batch_final.json --from $TIPPER_ACCOUNT --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYER_HOME --node $RPC_NODE --yes >> $WORKING_DIR/tx_batch_final_signed.json
    
    # Get the signed transaction file
    local signed_tx="$WORKING_DIR/tx_batch_final_signed.json"
    
    # Broadcast the signed transaction with sequence and account number
    log_message "Broadcasting signed batched transaction with sequence $ACCOUNT_SEQUENCE and account number $ACCOUNT_NUMBER"
    execute_with_retry \
        "./layerd tx broadcast $signed_tx --from $TIPPER_ACCOUNT --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYER_HOME --node $RPC_NODE --sequence $ACCOUNT_SEQUENCE --account-number $ACCOUNT_NUMBER --yes" \
        "Broadcasting batched transaction"
    
    log_message "Completed rewards distribution for block $block_height using batched transaction"
}

# Function to create batched transaction for a block
create_batched_transaction() {
    local block_height=$1
    
    log_message "Creating batched transaction for block $block_height"
    
    # Get the directory where layerd is located (current working directory when script runs)
    local WORKING_DIR=$(pwd)
    log_message "Working directory: $WORKING_DIR"
    
    # Clean up any existing transaction files
    rm -f "$WORKING_DIR"/tx_batch_*.json
    
    # Generate first transaction (time rewards)
    log_message "Generating time rewards transaction"
    ./layerd tx bank send $TIPPER_ACCOUNT $TIME_REWARDS_POOL ${TIME_REWARDS_AMOUNT}loya \
        --from $TIPPER_ACCOUNT \
        --chain-id $CHAIN_ID \
        --fees $FEES \
        --generate-only > "$WORKING_DIR"/tx_batch_time_rewards.json
    
    # Generate second transaction (validator fees)
    log_message "Generating validator fee transaction"
    ./layerd tx bank send $TIPPER_ACCOUNT $VALIDATOR_FEE_COLLECTOR ${VALIDATOR_FEE_AMOUNT}loya \
        --from $TIPPER_ACCOUNT \
        --chain-id $CHAIN_ID \
        --fees $FEES \
        --generate-only > "$WORKING_DIR"/tx_batch_validator_fees.json
    
    # Create batched transaction by combining both messages
    log_message "Creating batched transaction with both messages"
    
    # Extract the message from the second transaction and add it to the first
    local second_message=$(jq '.body.messages[0]' "$WORKING_DIR"/tx_batch_validator_fees.json)
    
    # Add the second message to the first transaction's messages array
    jq --argjson msg "$second_message" '.body.messages += [$msg]' "$WORKING_DIR"/tx_batch_time_rewards.json > "$WORKING_DIR"/tx_batch_combined.json
    
    # Copy the combined file to the final name
    cp "$WORKING_DIR"/tx_batch_combined.json "$WORKING_DIR"/tx_batch_final.json
    
    
    log_message "Batched transaction created: tx_batch_final.json"
    log_message "This transaction contains both:"
    log_message "  - ${TIME_REWARDS_AMOUNT}loya to time rewards pool"
    log_message "  - ${VALIDATOR_FEE_AMOUNT}loya to validator fee collector"
    log_message "Total fees: ${FEES}"
}

# Function to get account sequence and account number
get_account_info() {
    local account_address=$1
    
    log_message "Querying account info for $account_address"
    
    # Query the auth account
    local account_info
    account_info=$(./layerd query auth account $account_address --node https://node-palmito.tellorlayer.com/rpc/ --output json 2>/dev/null)
    
    if [ $? -ne 0 ] || [ -z "$account_info" ]; then
        log_message "Error: Failed to query account info for $account_address"
        return 1
    fi
    
    # Extract sequence and account number
    local sequence
    local account_number
    
    sequence=$(echo "$account_info" | jq -r '.account.value.sequence // .account.sequence // "0"')
    account_number=$(echo "$account_info" | jq -r '.account.value.account_number // .account.account_number // "0"')
    
    # Validate that we got valid values
    if [ "$sequence" = "null" ] || [ "$account_number" = "null" ] || [ -z "$sequence" ] || [ -z "$account_number" ]; then
        log_message "Error: Failed to get valid account info. Raw response:"
        log_message "$account_info"
        return 1
    fi
    
    log_message "Account sequence: $sequence, Account number: $account_number"
    
    # Return values through global variables
    ACCOUNT_SEQUENCE=$sequence
    ACCOUNT_NUMBER=$account_number
}





# Function to manually execute a batched transaction
execute_batched_transaction() {
    local block_height=$1
    
    log_message "Executing batched transaction for block $block_height"
    
    # Get account sequence and account number
    if ! get_account_info "$TIPPER_ACCOUNT"; then
        log_message "Error: Failed to get account info"
        return 1
    fi
    
    # Create the batched transaction
    create_batched_transaction "$block_height"
    
    # Sign the transaction
    log_message "Signing batched transaction"
    local WORKING_DIR=$(pwd)
    
    if ! ./layerd tx sign $WORKING_DIR/tx_batch_final.json --from $TIPPER_ACCOUNT --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYER_HOME --node $RPC_NODE --yes; then
        log_message "Error: Failed to sign batched transaction"
        return 1
    fi
    
    # Get the signed transaction file
    local signed_tx="$WORKING_DIR/tx_batch_final_signed.json"
    
    # Broadcast the signed transaction with sequence and account number
    log_message "Broadcasting signed batched transaction with sequence $ACCOUNT_SEQUENCE and account number $ACCOUNT_NUMBER"
    if ! ./layerd tx broadcast $signed_tx --from $TIPPER_ACCOUNT --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYER_HOME --node $RPC_NODE --sequence $ACCOUNT_SEQUENCE --account-number $ACCOUNT_NUMBER --yes; then
        log_message "Error: Failed to broadcast batched transaction"
        return 1
    fi
    
    log_message "Successfully executed batched transaction for block $block_height"
}

# Main execution loop
main() {
    log_message "Starting main execution loop"
    
    # Get initial block height
    local current_block=$(get_current_block)
    log_message "Starting from block $current_block"
    
    while true; do
        # Send rewards for current block
        if ! send_block_rewards "$current_block"; then
            log_message "Error occurred during execution, waiting 10 seconds before retrying..."
            sleep 1
            continue
        fi
        
        # Wait for next block
        if ! wait_for_next_block "$current_block"; then
            log_message "Error waiting for next block, retrying..."
            continue
        fi
        
        # Update current block
        if ! current_block=$(get_current_block); then
            log_message "Error: Failed to get current block, retrying..."
            sleep 5
            continue
        fi
        
        log_message "Successfully completed block $((current_block - 1)), moving to block $current_block"
    done
}

# Function to run in continuous mode (every block)
run_continuous() {
    log_message "Running in continuous mode - sending rewards every block"
    main
}

# Function to run for a specific number of blocks
run_for_blocks() {
    local num_blocks=$1
    log_message "Running for $num_blocks blocks"
    
    local current_block=$(get_current_block)
    local target_block=$((current_block + num_blocks))
    
    while [ "$current_block" -lt "$target_block" ]; do
        if ! send_block_rewards "$current_block"; then
            log_message "Error occurred, retrying..."
            sleep 5
            continue
        fi
        
        if ! wait_for_next_block "$current_block"; then
            log_message "Error waiting for next block, retrying..."
            sleep 5
            continue
        fi
        
        if ! current_block=$(get_current_block); then
            log_message "Error: Failed to get current block, retrying..."
            sleep 5
            continue
        fi
    done
    
    log_message "Completed rewards distribution for $num_blocks blocks"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -c, --continuous    Run continuously, sending rewards every block"
    echo "  -n, --num-blocks N  Run for N blocks then exit"
    echo "  -b, --batch         Create batched transaction files for current block"
    echo "  -e, --execute       Create, sign, and broadcast batched transaction for current block"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --continuous                    # Run continuously"
    echo "  $0 --num-blocks 10                # Run for 10 blocks"
    echo "  $0 --batch                        # Create batched transaction files"
    echo "  $0 --execute                       # Execute batched transaction immediately"
    echo ""
    echo "Note: Make sure to update the account addresses in the script before running"
}

# Parse command line arguments
case "${1:-}" in
    -c|--continuous)
        run_continuous
        ;;
    -n|--num-blocks)
        if [ -z "$2" ]; then
            echo "Error: Number of blocks required"
            show_usage
            exit 1
        fi
        run_for_blocks "$2"
        ;;
    -b|--batch)
        current_block=$(get_current_block)
        create_batched_transaction "$current_block"
        ;;
    -e|--execute)
        current_block=$(get_current_block)
        execute_batched_transaction "$current_block"
        ;;
    -h|--help|"")
        show_usage
        ;;
    *)
        echo "Unknown option: $1"
        show_usage
        exit 1
        ;;
esac
