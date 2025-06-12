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

# Find the closest snapshot height
# Snapshots are every 12500 blocks with remainder 10000 when divided by 12500
SNAPSHOT_INTERVAL=12500
SNAPSHOT_OFFSET=10000

remainder=$((CURRENT_HEIGHT % SNAPSHOT_INTERVAL))

if [ $remainder -ge $SNAPSHOT_OFFSET ]; then
    # We're past the snapshot point in this cycle, use current cycle
    ESTIMATED_SNAPSHOT_HEIGHT=$((CURRENT_HEIGHT - (remainder - SNAPSHOT_OFFSET) - 12500))
else
    # We're before the snapshot point, use previous cycle
    ESTIMATED_SNAPSHOT_HEIGHT=$((CURRENT_HEIGHT - (remainder + SNAPSHOT_INTERVAL - SNAPSHOT_OFFSET) - 12500))
fi

export TRUSTED_HASH=$(curl -s "https://node-palmito.tellorlayer.com/rpc/block?height=$ESTIMATED_SNAPSHOT_HEIGHT" | jq -r .result.block_id.hash) && echo $TRUSTED_HASH

echo -e "\n"
echo "Node URL: $NODE_URL"
echo "Node ID: $NODE_ID"
echo "Current height: $CURRENT_HEIGHT"
echo "Likely snapshot height: $ESTIMATED_SNAPSHOT_HEIGHT"
echo "Trusted hash: $TRUSTED_HASH"

echo -e "\n"
read -p "Press enter to continue or ctrl+c to exit"
clear

# create statesync configuration for snapshot node
sudo sed -i '' "s|^rpc_servers = .*|rpc_servers = \"$NODE_URL,$NODE_URL\"|" $HOME_DIR/config/config.toml
sudo sed -i '' "s|^trust_height = .*|trust_height = $ESTIMATED_SNAPSHOT_HEIGHT|" $HOME_DIR/config/config.toml
sudo sed -i '' "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" $HOME_DIR/config/config.toml
sudo sed -i '' "s|^persistent_peers = .*|persistent_peers = \"$NODE_ID@node-palmito.tellorlayer.com:26656\"|" $HOME_DIR/config/config.toml

echo -e "\n"
echo "Verify StateSync configuration"
cat $HOME_DIR/config/config.toml | awk '/rpc_servers|trust_height|trust_hash|persistent_peers/ && !/experimental/'

echo -e "\n"
read -p "press enter to start the node, or ctrl+c to exit"
clear

./layerd start --home ~/.layer --keyring-backend test --key-name test --api.enable --api.swagger
