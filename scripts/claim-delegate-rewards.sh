#!/bin/bash

# Set up logging
LOG_FILE="/var/log/claim-delegate-rewards.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Function to log messages
log_message() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

# clear the terminal
clear

# Stop execution if any command fails
set -e

log_message "Starting claim-delegate-rewards script"

export KEYRING_BACKEND="test"

export DELEGATOR_ONE_ADD="tellor1zhg69su8p5zplr7jkzkav7874ekncn83592rlk"
export DELEGATOR_TWO_ADD="tellor1hu8vk2zzyety0j4c88vpyfew8rc7frqnjf63nm"
export DELEGATOR_THREE_ADD="tellor1p7y0yvqnw3ajjq6vsjv7yahsvjxs6ahhs5lxq7"
export DELEGATOR_FOUR_ADD="tellor10zjsg0205re8g74t6qqqd0jh4nm8cuyy4nw99g"
export DELEGATOR_FIVE_ADD="tellor1uasku64eztzzwne8gx58kzxkfn5lu803ekfqpn"
export DELEGATOR_SIX_ADD="tellor1edr39pfjd2j0l7335dx2zkgk5zmtl5cwnuh6xw"

export VALIDATOR_ONE_ADD="tellorvaloper168hv5trkskdwvxj2hzqdmch7r7l0prlnvwud6j"
export VALIDATOR_TWO_ADD="tellorvaloper1fyc97jykpyvwrm78f0gfvffg987sqlvuwpzc8k"
export VALIDATOR_THREE_ADD="tellorvaloper12mtrv5yn54xqsvdfvkz0g92w8wmeqyf3lenjxy"
export VALIDATOR_FOUR_ADD="tellorvaloper1gtrkqyyzwhag05jl87h5d75khv4s6r2xml7zcu"
export VALIDATOR_FIVE_ADD="tellorvaloper1uac7nf2wgq203phmh00lvu2pkwj2rz294gckyj"
export VALIDATOR_SIX_ADD="tellorvaloper16vuuwx7lekkfy57nl9mxmxrplfzflgylndyh7w"

export EXECUTING_ADDRESS="tellor168hv5trkskdwvxj2hzqdmch7r7l0prlnepslrz"

# Function to execute transaction with retry
execute_with_retry() {
    local msg_file=$1
    local max_attempts=5
    local attempt=1
    local success=false

    while [ $attempt -le $max_attempts ] && [ "$success" = false ]; do
        log_message "Attempt $attempt of $max_attempts for $msg_file"
        
        # Capture the output of the command
        local output
        output=$(./layerd tx authz exec $msg_file --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --yes 2>&1)
        
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
            log_message "Transaction successful for $msg_file"
        else
            log_message "Transaction failed for $msg_file, attempt $attempt of $max_attempts"
            log_message "Error: $raw_log"
            if [ $attempt -lt $max_attempts ]; then
                log_message "Waiting 5 seconds before retry..."
                sleep 5
            fi
        fi
        attempt=$((attempt + 1))
    done

    if [ "$success" = false ]; then
        log_message "Failed to execute transaction for $msg_file after $max_attempts attempts"
        return 1
    fi
}

# Generate individual withdraw-tip messages
log_message "Generating withdraw-tip messages"
./layerd tx reporter withdraw-tip $DELEGATOR_ONE_ADD $VALIDATOR_ONE_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_1.json
./layerd tx reporter withdraw-tip $DELEGATOR_TWO_ADD $VALIDATOR_TWO_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_2.json
./layerd tx reporter withdraw-tip $DELEGATOR_THREE_ADD $VALIDATOR_THREE_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_3.json
./layerd tx reporter withdraw-tip $DELEGATOR_FOUR_ADD $VALIDATOR_FOUR_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_4.json
./layerd tx reporter withdraw-tip $DELEGATOR_FIVE_ADD $VALIDATOR_FIVE_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_5.json
./layerd tx reporter withdraw-tip $DELEGATOR_SIX_ADD $VALIDATOR_SIX_ADD --from $EXECUTING_ADDRESS --chain-id tellor-1 --fees 20loya --generate-only > msg_withdraw_tip_6.json

# Execute each transaction with retry logic
log_message "Executing transactions"
execute_with_retry msg_withdraw_tip_1.json
sleep 2
execute_with_retry msg_withdraw_tip_2.json
sleep 2
execute_with_retry msg_withdraw_tip_3.json
sleep 2
execute_with_retry msg_withdraw_tip_4.json
sleep 2
execute_with_retry msg_withdraw_tip_5.json
sleep 2
execute_with_retry msg_withdraw_tip_6.json

# Clean up temporary files
log_message "Cleaning up temporary files"
rm msg_withdraw_tip_*.json

log_message "Script completed successfully"






