#!/bin/bash

PRE_UPGRADE_BRANCH="tags/v5.1.0"
UPGRADE_BRANCH="feat/v5.1.1-upgrade-handler"
UPGRADE_NAME="v5.1.1"

# Function to log messages
log_message() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

# Function to execute transaction with retry
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

echo "Please make sure your git repo is in a clean state before running this script"
echo "We will be switching branches and uncommitted changes could result in errors"

echo "Switching to $PRE_UPGRADE_BRANCH branch"
git checkout $PRE_UPGRADE_BRANCH

echo "starting devnet from $PRE_UPGRADE_BRANCH branch"
bash ./run_current_branch_devnet.sh

echo "building layerd binary for tx's called using the local layerd binary"
go build -o ../../layerd ../../cmd/layerd

echo "Create upgrade proposal json"
#Create upgrade proposal JSON
cat > proposal.json << EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx",
      "plan": {
        "name": "$UPGRADE_NAME",
        "time": "0001-01-01T00:00:00Z",
        "height": "300",
        "info": "Upgrade to $UPGRADE_NAME",
        "upgraded_client_state": null
      }
    }
  ],
  "metadata": "ipfs://CID",
  "deposit": "100000loya",
  "title": "$UPGRADE_NAME",
  "summary": "Upgrade to $UPGRADE_NAME",
  "expedited": true
}
EOF

echo "Submit the upgrade proposal"
# Submit the upgrade proposal using the execute_with_retry function
upgrade_cmd="../../layerd tx gov submit-proposal ./proposal.json --from validator-0 --chain-id tellor-devnet --keyring-backend test --keyring-dir ./validator-info/validator-0 --fees 25loya --yes"
execute_with_retry "$upgrade_cmd"

# Capture the current time when proposal is submitted
PROPOSAL_SUBMIT_TIME=$(date +%s)
echo "Proposal submitted at timestamp: $PROPOSAL_SUBMIT_TIME"
sleep 3

echo "vote for upgrade proposal with all validators"
for i in {0..2}
do
    echo "Voting with validator $i"
    vote_cmd="../../layerd tx gov vote 1 yes --from validator-$i --chain-id tellor-devnet --keyring-backend test --keyring-dir ./validator-info/validator-$i --fees 25loya --yes"
    execute_with_retry "$vote_cmd"
done

echo "query the tally of the votes for the upgrade proposal"
../../layerd query gov tally 1
sleep 1

echo "checkout to $UPGRADE_BRANCH branch to build the upgrade binary"
git checkout $UPGRADE_BRANCH


echo "create a new branch for the upgrade"
git checkout -b $UPGRADE_BRANCH

echo "build the upgrade binary"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ../../layerd_upgrade ../../cmd/layerd

echo "Now that the upgrade binary is built. Monitor the logs of validator-0 using this command: docker logs -f validator-node-0"

# Wait for proposal to pass
echo "Waiting for proposal voting period to end (300 seconds)..."
while true; do
    # Calculate time elapsed since proposal submission
    CURRENT_TIME=$(date +%s)
    TIME_ELAPSED=$((CURRENT_TIME - PROPOSAL_SUBMIT_TIME))
    
    # Query the proposal status
    PROPOSAL_STATUS=$(../../layerd query gov proposal 1 --output json | jq -r '.proposal.status')
    
    # If we've passed the voting period (300 seconds) and status is not 2 (voting period)
    if [ $TIME_ELAPSED -ge 300 ] && [ "$PROPOSAL_STATUS" != "2" ]; then
        if [ "$PROPOSAL_STATUS" = "3" ]; then
            echo "Proposal has passed successfully!"
            break
        else
            echo "Proposal did not pass. Final status: $PROPOSAL_STATUS"
            exit 1
        fi
    fi
    
    echo "Time elapsed since proposal submission: ${TIME_ELAPSED} seconds"
    echo "Current proposal status: $PROPOSAL_STATUS"
    
    # Wait 10 seconds before checking again
    sleep 10
done

echo "Monitoring validator-0 logs for upgrade message..."
while true; do
    if docker logs validator-node-0 2>&1 | grep -q "err=\"failed to apply block; error UPGRADE \"$UPGRADE_NAME\" NEEDED\""; then
        echo "Upgrade message detected! Proceeding with upgrade..."
        break
    fi
    sleep 5
done

echo "Stopping containers..."
docker compose down

echo "Replacing layerd binary with upgrade version..."
cp ../../layerd_upgrade ./bin/layerd

echo "Starting containers with new binary..."
docker compose up -d

echo "Upgrade process completed. New version is now running."





