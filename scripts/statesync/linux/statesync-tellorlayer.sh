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
export HOME_DIR=/home/$(logname)/.layer
export TEMP_LOG_FILE="/home/$(logname)/layerd_statesync.log"

echo "Debug: TEMP_LOG_FILE will be created at: $TEMP_LOG_FILE"

# set statesync enable = true
sudo sed -i "s|^enable = .*|enable = true|" $HOME_DIR/config/config.toml

# set configs so temporary node will start
export TRUSTED_HEIGHT=$CURRENT_HEIGHT
sed -i "s|^trust_height = .*|trust_height = $TRUSTED_HEIGHT|" $HOME_DIR/config/config.toml
export TRUSTED_HASH=$(curl -s "https://node-palmito.tellorlayer.com/rpc/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
sed -i "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" $HOME_DIR/config/config.toml
sudo sed -i "s|^persistent_peers = .*|persistent_peers = \"$NODE_ID@node-palmito.tellorlayer.com:26656\"|" $HOME_DIR/config/config.toml

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

# Extract just the height numbers from the snapshot lines using awk and strip ANSI codes
SNAPSHOT_HEIGHTS=$(echo "$SNAPSHOT_LINES" | awk -F'height=' '{print $2}' | awk '{print $1}' | sed 's/\x1b\[[0-9;]*m//g')
echo "Debug: Found snapshot heights:"
echo "$SNAPSHOT_HEIGHTS"
echo ""

# get the highest/newest snapshot height
EXACT_TRUSTED_HEIGHT=$(echo "$SNAPSHOT_HEIGHTS" | sort -n | tail -1)
echo "Debug: Selected height: $EXACT_TRUSTED_HEIGHT"

# get the trusted hash for the exact height
if [ -n "$EXACT_TRUSTED_HEIGHT" ] && [ "$EXACT_TRUSTED_HEIGHT" != "" ]; then
    echo "Getting trusted hash for height $EXACT_TRUSTED_HEIGHT..."
    export EXACT_TRUSTED_HASH=$(curl -s "https://node-palmito.tellorlayer.com/rpc/block?height=$EXACT_TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
    echo "Debug: Retrieved hash: $EXACT_TRUSTED_HASH"
else
    echo "Warning: No valid snapshot height found, using fallback method"
    EXACT_TRUSTED_HEIGHT=$((CURRENT_HEIGHT - 1000))
    export EXACT_TRUSTED_HASH=$(curl -s "https://node-palmito.tellorlayer.com/rpc/block?height=$EXACT_TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
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
sudo sed -i "s|^rpc_servers = .*|rpc_servers = \"$NODE_URL,$NODE_URL\"|" $HOME_DIR/config/config.toml
sudo sed -i "s|^trust_height = .*|trust_height = $EXACT_TRUSTED_HEIGHT|" $HOME_DIR/config/config.toml
sudo sed -i "s|^trust_hash = .*|trust_hash = \"$EXACT_TRUSTED_HASH\"|" $HOME_DIR/config/config.toml

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
