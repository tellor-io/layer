#!/bin/bash

echo "---------------------------------------------------"
echo ""
echo "This script will clear all chain data from your local layer node and resync the chain."
echo "Make sure your node is stopped before running this script!"
echo "Your configurations and accounts will be preserved!"
echo ""
echo "---------------------------------------------------"
read -p "Press enter to continue or ctrl+c to exit"

# clear all data from the layer node
rm -rf ~/.layer/data/application.db 
rm -rf ~/.layer/data/blockstore.db
rm -rf ~/.layer/data/cs.wal
rm -rf ~/.layer/data/evidence.db
rm -rf ~/.layer/data/snapshots
rm -rf ~/.layer/data/state.db
rm -rf ~/.layer/data/tx_index.db

export NODE_URL="https://mainnet.tellorlayer.com/rpc"
export CURRENT_HEIGHT=$(./layerd status --node $NODE_URL | jq -r '.sync_info.latest_block_height')
export NODE_ID=$(./layerd status --node $NODE_URL | jq -r '.node_info.id')
export HOME_DIR="/home/$(logname)/.layer"
export TEMP_LOG_FILE="/home/$(logname)/layerd_statesync.log"
export PEERS="5a9db46eceb055c9238833aa54e15a2a32a09c9a@54.67.36.145:26656,f2644778a8a2ca3b55ec65f1b7799d32d4a7098e@54.149.160.93:26656,2904aa32501548e127d3198c8f5181fb4d67bbe6@18.116.23.104:26656"
export KEY_NAME="test"

echo "Debug: TEMP_LOG_FILE will be created at: $TEMP_LOG_FILE"

# set statesync enable = true
sed -i "s|^enable = .*|enable = true|" $HOME_DIR/config/config.toml

# set configs so temporary node will start
export TRUSTED_HEIGHT=$CURRENT_HEIGHT
sed -i "s|^trust_height = .*|trust_height = $TRUSTED_HEIGHT|" $HOME_DIR/config/config.toml
export TRUSTED_HASH=$(curl -s "$NODE_URL/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
sed -i "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" $HOME_DIR/config/config.toml
sed -i "s|^persistent_peers = .*|persistent_peers = \"$PEERS\"|" $HOME_DIR/config/config.toml

# Start layerd in background and capture logs
echo "Starting layerd to discover snapshots..."
./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger > $TEMP_LOG_FILE 2>&1 &
LAYERD_PID=$!

# wait for node to start and discover snapshots
echo "Waiting for node to discover snapshots..."
sleep 5

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

# Parse snapshot lines to extract heights, then find the highest height
if [ -n "$SNAPSHOT_LINES" ]; then
    # Create a temporary file to store heights
    TEMP_SNAPSHOTS=$(mktemp)
    
    # Process each line and extract heights only
    while IFS= read -r line; do
        # Extract height from each line, stripping ANSI codes
        HEIGHT=$(echo "$line" | awk -F'height=' '{print $2}' | awk '{print $1}' | sed 's/\x1b\[[0-9;]*m//g')
        if [ -n "$HEIGHT" ] && [[ "$HEIGHT" =~ ^[0-9]+$ ]]; then
            echo "$HEIGHT" >> "$TEMP_SNAPSHOTS"
        fi
    done <<< "$SNAPSHOT_LINES"
    
    # Find the second-highest height
    if [ -s "$TEMP_SNAPSHOTS" ]; then
        EXACT_TRUSTED_HEIGHT=$(sort -nr "$TEMP_SNAPSHOTS" | sed -n '2p')
        
        echo "Debug: Selected second-highest snapshot height: $EXACT_TRUSTED_HEIGHT"
        
        # Clean up temp file
        rm -f "$TEMP_SNAPSHOTS"
        
        # Verify we got a valid height
        if [ -z "$EXACT_TRUSTED_HEIGHT" ] || ! [[ "$EXACT_TRUSTED_HEIGHT" =~ ^[0-9]+$ ]]; then
            echo "Error: Failed to extract valid height from snapshot logs. Exiting."
            exit 1
        fi
        
        # Now get the trusted hash for this height using block query
        echo "Querying block hash for height $EXACT_TRUSTED_HEIGHT..."
        EXACT_TRUSTED_HASH=$(curl -s "$NODE_URL/block?height=$EXACT_TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
        
        if [ -z "$EXACT_TRUSTED_HASH" ] || [ "$EXACT_TRUSTED_HASH" = "null" ]; then
            echo "Error: Failed to get valid hash for height $EXACT_TRUSTED_HEIGHT. Exiting."
            exit 1
        fi
        
        echo "Debug: Retrieved trusted hash: $EXACT_TRUSTED_HASH"
    else
        rm -f "$TEMP_SNAPSHOTS"
        echo "Error: No valid heights found in snapshot logs."
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

echo "Configuration Complete!"

# Check if user wants to start the node now
echo "--------------------------------"
echo ""
echo "Do you want to start the layer node now?"
echo "1) Yes, start the node now"
echo "2) No, I'll start it manually later"
echo "--------------------------------"
read -p "Select an option [1-2]: " start_node_choice

case "$start_node_choice" in
  1)
    echo "Starting layer node..."
    echo "Note: The node will run in the foreground. Press Ctrl+C to stop."
    echo "Starting in 3 seconds..."
    sleep 3
    ./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger
    ;;
  2)
    echo "Node startup skipped."
    echo "To start the node later, run:"
    echo "./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger"
    ;;
  *)
    echo "Invalid option. Node startup skipped."
    echo "To start the node later, run:"
    echo "./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger"
    ;;
esac
