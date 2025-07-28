#!/bin/bash

echo "This script will clear all chain data from your local layer node and resync the chain."
echo "Your configurations and accounts will be preserved!"
read -p "Press enter to continue or ctrl+c to exit"

# clear all data from the layer node
rm -rf ~/.layer/data/application.db 
rm -rf ~/.layer/data/blockstore.db
rm -rf ~/.layer/data/cs.wal
rm -rf ~/.layer/data/evidence.db
rm -rf ~/.layer/data/snapshots
rm -rf ~/.layer/data/state.db
rm -rf ~/.layer/data/tx_index.db

export NODE_URL="https://node-palmito.tellorlayer.com/rpc/"
export CURRENT_HEIGHT=$(./layerd status --node $NODE_URL | jq -r '.sync_info.latest_block_height')
export NODE_ID=$(./layerd status --node $NODE_URL | jq -r '.node_info.id')
export HOME_DIR="/home/spuddy/.layer"
export TEMP_LOG_FILE="/home/spuddy/layerd_statesync.log"

echo "Debug: TEMP_LOG_FILE will be created at: $TEMP_LOG_FILE"

# set statesync enable = true
sed -i "s|^enable = .*|enable = true|" $HOME_DIR/config/config.toml

# set configs so temporary node will start
export TRUSTED_HEIGHT=$CURRENT_HEIGHT
sed -i "s|^trust_height = .*|trust_height = $TRUSTED_HEIGHT|" $HOME_DIR/config/config.toml
export TRUSTED_HASH=$(curl -s "https://node-palmito.tellorlayer.com/rpc/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
sed -i "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" $HOME_DIR/config/config.toml
sed -i "s|^persistent_peers = .*|persistent_peers = \"$NODE_ID@node-palmito.tellorlayer.com:26656\"|" $HOME_DIR/config/config.toml

# Start layerd in background and capture logs
echo "Starting layerd to discover snapshots..."
./layerd start --home ~/.layer --keyring-backend test --key-name test --api.enable --api.swagger > $TEMP_LOG_FILE 2>&1 &
LAYERD_PID=$!

# wait for node to start and discover snapshots
echo "Waiting for node to discover snapshots..."
sleep 3

# Check if log file exists and has content
if [ ! -f "$TEMP_LOG_FILE" ]; then
    echo "Error: Log file $TEMP_LOG_FILE was not created!"
    kill $LAYERD_PID 2>/dev/null
    exit 1
fi

# search the logs for the best snapshot height to use
SNAPSHOT_LINES=$(grep "Discovered new snapshot" "$TEMP_LOG_FILE")
echo "Debug: Found snapshot lines:"
echo "$SNAPSHOT_LINES"
echo ""

# Parse snapshot lines to extract height and hash pairs, then find the highest height
if [ -n "$SNAPSHOT_LINES" ]; then
    # Create a temporary file to store height:hash pairs
    TEMP_SNAPSHOTS=$(mktemp)
    
    # Process each line and extract height:hash pairs
    while IFS= read -r line; do
        # Extract height and hash from each line, stripping ANSI codes
        HEIGHT=$(echo "$line" | awk -F'height=' '{print $2}' | awk '{print $1}' | sed 's/\x1b\[[0-9;]*m//g')
        HASH=$(echo "$line" | awk -F'hash=' '{print $2}' | awk '{print $1}' | sed 's/\x1b\[[0-9;]*m//g')
        if [ -n "$HEIGHT" ] && [ -n "$HASH" ]; then
            echo "$HEIGHT:$HASH" >> "$TEMP_SNAPSHOTS"
        fi
    done <<< "$SNAPSHOT_LINES"
    
    # Find the line with the highest height
    if [ -s "$TEMP_SNAPSHOTS" ]; then
        HIGHEST_SNAPSHOT=$(sort -n "$TEMP_SNAPSHOTS" | tail -1)
        EXACT_TRUSTED_HEIGHT=$(echo "$HIGHEST_SNAPSHOT" | cut -d':' -f1)
        EXACT_TRUSTED_HASH=$(echo "$HIGHEST_SNAPSHOT" | cut -d':' -f2)
        
        echo "Debug: Selected highest snapshot - Height: $EXACT_TRUSTED_HEIGHT, Hash: $EXACT_TRUSTED_HASH"
        
        # Clean up temp file
        rm -f "$TEMP_SNAPSHOTS"
        
        # Verify we got valid values
        if [ -z "$EXACT_TRUSTED_HEIGHT" ] || [ -z "$EXACT_TRUSTED_HASH" ] || [ "$EXACT_TRUSTED_HASH" = "null" ]; then
            echo "Error: Failed to extract valid height and hash from snapshot logs. Exiting."
            exit 1
        fi
    else
        rm -f "$TEMP_SNAPSHOTS"
        echo "Error: No valid height:hash pairs found in snapshot logs."
        exit 1
    fi
else
    echo "Error: No snapshots discovered. Cannot proceed with statesync without a valid snapshot height."
    echo "This means either:"
    echo "  1. The node is not serving snapshots"
    echo "  2. The node failed to start properly"
    echo "  3. No snapshot logs were found in the output"
    exit 1
fi

# Stop the temporary layerd process
# this will also clear the data from the layer node
echo "Stopping temporary layerd process..."
kill $LAYERD_PID 2>/dev/null
wait $LAYERD_PID 2>/dev/null

# clear all data from the layer node again :)
rm -rf ~/.layer/data/application.db 
rm -rf ~/.layer/data/blockstore.db
rm -rf ~/.layer/data/cs.wal
rm -rf ~/.layer/data/evidence.db
rm -rf ~/.layer/data/snapshots
rm -rf ~/.layer/data/state.db
rm -rf ~/.layer/data/tx_index.db

# set trusted height and hash again with height that will work
sed -i "s|^trust_height = .*|trust_height = $EXACT_TRUSTED_HEIGHT|" $HOME_DIR/config/config.toml
sed -i "s|^trust_hash = .*|trust_hash = \"$EXACT_TRUSTED_HASH\"|" $HOME_DIR/config/config.toml

echo -e "\n"
echo "====== StateSync Configuration ======"
echo "RPC Server: $NODE_URL"
echo "Current Height: $CURRENT_HEIGHT"
echo "Exact Trusted Height: $EXACT_TRUSTED_HEIGHT"
echo "Exact Trusted Hash: $EXACT_TRUSTED_HASH"
echo "Node ID: $NODE_ID"
echo -e "\nConfiguration file entries:"
cat $HOME_DIR/config/config.toml | awk '/rpc_servers|trust_height|trust_hash|persistent_peers/ && !/experimental/'

echo -e "\n"
read -p "press enter to start the node, or ctrl+c to exit"
clear

# Clean up temporary log file
rm -f $TEMP_LOG_FILE

./layerd start --home ~/.layer --keyring-backend test --key-name test --api.enable --api.swagger
