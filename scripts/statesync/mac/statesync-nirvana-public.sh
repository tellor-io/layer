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

export NODE_URL="https://tellor-testnet.nirvanalabs.xyz/tellor-testnet-public"
export CURRENT_HEIGHT=$(./layerd status --node $NODE_URL | jq -r '.sync_info.latest_block_height')
export NODE_ID=$(./layerd status --node $NODE_URL | jq -r '.node_info.id')
export HOME_DIR=/home/$(logname)/.layer

# Find the closest snapshot height using the pattern:
# Base: 3356000, then +398000, then +10000 (twice), then +12000 (repeating)
BASE_HEIGHT=3356000
PHASE1_JUMP=398000    # First jump to 3754000
PHASE2_INTERVAL=10000 # Used twice (3764000, 3774000)  
PHASE3_INTERVAL=12000 # Repeating interval after 3774000

# Calculate which snapshot we should use (always use the previous one for stability)
if [ $CURRENT_HEIGHT -le $BASE_HEIGHT ]; then
    # At or before first snapshot - use base
    ESTIMATED_SNAPSHOT_HEIGHT=$BASE_HEIGHT
elif [ $CURRENT_HEIGHT -le $((BASE_HEIGHT + PHASE1_JUMP)) ]; then
    # In first jump range - use base
    ESTIMATED_SNAPSHOT_HEIGHT=$BASE_HEIGHT
elif [ $CURRENT_HEIGHT -le $((BASE_HEIGHT + PHASE1_JUMP + PHASE2_INTERVAL)) ]; then
    # In first phase2 interval - use previous (3754000)
    ESTIMATED_SNAPSHOT_HEIGHT=$((BASE_HEIGHT + PHASE1_JUMP))
elif [ $CURRENT_HEIGHT -le $((BASE_HEIGHT + PHASE1_JUMP + 2 * PHASE2_INTERVAL)) ]; then
    # In second phase2 interval - use previous (3764000)
    ESTIMATED_SNAPSHOT_HEIGHT=$((BASE_HEIGHT + PHASE1_JUMP + PHASE2_INTERVAL))
else
    # In phase3 (repeating 12000 intervals) - use previous iteration
    PHASE3_START=$((BASE_HEIGHT + PHASE1_JUMP + 2 * PHASE2_INTERVAL))
    HEIGHT_IN_PHASE3=$((CURRENT_HEIGHT - PHASE3_START))
    PHASE3_ITERATIONS=$((HEIGHT_IN_PHASE3 / PHASE3_INTERVAL))
    
    # Use previous snapshot: if we're in iteration N, use iteration N-1
    if [ $PHASE3_ITERATIONS -eq 0 ]; then
        # In first phase3 iteration, use last phase2 snapshot
        ESTIMATED_SNAPSHOT_HEIGHT=$((BASE_HEIGHT + PHASE1_JUMP + 2 * PHASE2_INTERVAL))
    else
        # Use previous phase3 iteration
        ESTIMATED_SNAPSHOT_HEIGHT=$((PHASE3_START + (PHASE3_ITERATIONS - 1) * PHASE3_INTERVAL))
    fi
fi

export TRUSTED_HASH=$(curl -s "https://tellor-testnet.nirvanalabs.xyz/tellor-testnet-public/block?height=$ESTIMATED_SNAPSHOT_HEIGHT" | jq -r .result.block_id.hash) && echo $TRUSTED_HASH

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
sudo sed -i '' "s|^persistent_peers = .*|persistent_peers = \"$NODE_ID@tellor-testnet.nirvanalabs.xyz/tellor-testnet-public:26656\"|" $HOME_DIR/config/config.toml

echo -e "\n"
echo "Verify StateSync configuration"
cat $HOME_DIR/config/config.toml | awk '/rpc_servers|trust_height|trust_hash|persistent_peers/ && !/experimental/'

echo -e "\n"
read -p "press enter to start the node, or ctrl+c to exit"
clear

./layerd start --home ~/.layer --keyring-backend test --key-name test --api.enable --api.swagger
