#!/bin/bash

export LAYER_SERVICE_NAME="layer"
export LAYERD_PATH="/home/ubuntu/layer/layerd"
export REPORTER_ADDRESS=""
export CHAIN_ID="tellor-devnet"
export KEYRING_BACKEND="test"
export KEYRING_DIR=""
export FEES="25loya"
export KEY_NAME=""

# Set up logging
LOG_FILE="/var/log/automated-chain-checks.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Function to log messages
log_message() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

execute_with_retry() {
    local cmd=$1
    local max_attempts=10
    local attempt=1
    local success=false

    while [ $attempt -le $max_attempts ] && [ "$success" = false ]; do
        log_message "Attempt $attempt of $max_attempts"
        
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
            log_message "Transaction successful"
        else
            log_message "Transaction failed, attempt $attempt of $max_attempts"
            log_message "Error: $raw_log"
            if [ $attempt -lt $max_attempts ]; then
                log_message "Waiting 5 seconds before retry..."
                sleep 5
            fi
        fi
        attempt=$((attempt + 1))
    done

    if [ "$success" = false ]; then
        log_message "Failed to execute transaction after $max_attempts attempts"
        return 1
    fi
}

sudo systemctl restart $LAYER_SERVICE_NAME

sleep 5

sudo systemctl status $LAYER_SERVICE_NAME

echo "report a spot price...."

# Example submit-value command using the same retry structure
# You can customize these variables based on your needs
QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
VALUE="000000000000000000000000000000000000000000000084bd26b6c2dd7c0000"

log_message "Submitting spot price value..."
submit_value_cmd="$LAYERD_PATH tx oracle submit-value $QUERY_DATA $VALUE --from $KEY_NAME --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --keyring-dir $KEYRING_DIR --fees $FEES --yes"
execute_with_retry "$submit_value_cmd"
log_message "Spot price submission completed"

# Prompt for transaction details
echo "Enter sender address:"
read SENDER_ADDRESS

echo "Enter recipient address:"
read RECIPIENT_ADDRESS

echo "Enter amount in loya:"
read AMOUNT_LOYAL

# Log the collected information
log_message "Sender address: $SENDER_ADDRESS"
log_message "Recipient address: $RECIPIENT_ADDRESS"
log_message "Amount: ${AMOUNT_LOYAL}loya"

# Execute bank send transaction using the collected variables
log_message "Executing bank send transaction..."
bank_send_cmd="$LAYERD_PATH tx bank send $SENDER_ADDRESS $RECIPIENT_ADDRESS ${AMOUNT_LOYAL}loya --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --keyring-dir $KEYRING_DIR --fees $FEES --yes"
execute_with_retry "$bank_send_cmd"
log_message "Bank send transaction completed"

echo "Enter a validator address to delegate tokens to: "
read VALIDATOR_ADDRESS

echo "Enter the amount of tokens to delegate: "
read DELEGATION_AMOUNT

log_message "Executing delegate transaction..."
delegate_cmd="$LAYERD_PATH tx staking delegate $VALIDATOR_ADDRESS $DELEGATION_AMOUNT --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --keyring-dir $KEYRING_DIR --fees $FEES --yes"
execute_with_retry "$delegate_cmd"
log_message "Delegate transaction completed"

# Query bank balance for reporter address
if [ -z "$REPORTER_ADDRESS" ]; then
    log_message "ERROR: REPORTER_ADDRESS is not set. Please set the reporter address variable."
    exit 1
fi

log_message "Querying bank balance for reporter address: $REPORTER_ADDRESS"
balance_output=$($LAYERD_PATH query bank balances $REPORTER_ADDRESS --output json 2>&1)

if [ $? -eq 0 ]; then
    # Extract the loya amount using jq (JSON processor)
    BEFORE_LOYA_BALANCE=$(echo "$balance_output" | jq -r '.balances[] | select(.denom == "loya") | .amount')
    
    if [ "$BEFORE_LOYA_BALANCE" != "null" ] && [ -n "$BEFORE_LOYA_BALANCE" ]; then
        log_message "Reporter address loya balance: $BEFORE_LOYA_BALANCE"
        log_message "Balance query successful"
    else
        log_message "No loya balance found for reporter address"
        BEFORE_LOYA_BALANCE="0"
    fi
else
    log_message "ERROR: Failed to query bank balance"
    log_message "Error output: $balance_output"
    BEFORE_LOYA_BALANCE="0"
fi

log_message "Claiming validator rewards from account..."
claim_rewards_cmd="$LAYERD_PATH tx distribution withdraw-all-rewards --from $KEY_NAME --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --keyring-dir $KEYRING_DIR --fees $FEES --yes"
execute_with_retry "$claim_rewards_cmd"
log_message "Validator rewards claimed"

log_message "Querying bank balance for reporter address again: $REPORTER_ADDRESS"
balance_output=$($LAYERD_PATH query bank balances $REPORTER_ADDRESS --output json 2>&1)

if [ $? -eq 0 ]; then
    # Extract the loya amount using jq (JSON processor)
    AFTER_LOYA_BALANCE=$(echo "$balance_output" | jq -r '.balances[] | select(.denom == "loya") | .amount')
    
    if [ "$BEFORE_LOYA_BALANCE" != "null" ] && [ -n "$AFTER_LOYA_BALANCE" ]; then
        log_message "Reporter address loya balance: $AFTER_LOYA_BALANCE"
        log_message "Balance query successful"
    else
        log_message "No loya balance found for reporter address"
        AFTER_LOYA_BALANCE="0"
    fi
else
    log_message "ERROR: Failed to query bank balance"
    log_message "Error output: $balance_output"
    AFTER_LOYA_BALANCE="0"
fi

# Compare before and after balances
log_message "Comparing balances..."
log_message "Before balance: $BEFORE_LOYA_BALANCE"
log_message "After balance: $AFTER_LOYA_BALANCE"

# Convert to integers for numerical comparison (remove any decimal points)
BEFORE_INT=$(echo "$BEFORE_LOYA_BALANCE" | sed 's/\..*//')
AFTER_INT=$(echo "$AFTER_LOYA_BALANCE" | sed 's/\..*//')

if [ "$BEFORE_INT" -gt "$AFTER_INT" ]; then
    log_message "Before balance ($BEFORE_LOYA_BALANCE) is greater than after balance ($AFTER_LOYA_BALANCE)"
elif [ "$AFTER_INT" -gt "$BEFORE_INT" ]; then
    log_message "After balance ($AFTER_LOYA_BALANCE) is greater than before balance ($BEFORE_LOYA_BALANCE)"
else
    log_message "Before balance ($BEFORE_LOYA_BALANCE) equals after balance ($AFTER_LOYA_BALANCE)"
fi













