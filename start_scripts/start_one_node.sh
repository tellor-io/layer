#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"

export LAYERD_NODE_HOME="$HOME/.layer/bill"

# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer/bill
rm -rf ~/.layer/config
rm -rf ~/.layer/data

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd


# Initialize chain node with the folder for bill
echo "Initializing chain node for bill..."
./layerd init billmoniker --chain-id layertest-4 --home ~/.layer/bill

# Add a validator account for bill
echo "Adding validator account for bill..."
./layerd keys add bill --keyring-backend test --home ~/.layer/bill

echo "Change denom to loya in genesis file..."
sed -i '' 's/"stake"/"loya"/g' ~/.layer/bill/config/genesis.json

echo "Change denom to loya in config files for bill..."
sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/bill/config/app.toml

echo "Set the keyring backend in client.toml to environment variable for bill..."
sed -i '' 's/^keyring-backend = .*"/keyring-backend = "test"/g' ~/.layer/bill/config/client.toml

echo "Set Chain Id to layer in client config file for bill..."
sed -i '' 's/^chain-id = .*$/chain-id = "layertest-4"/g' ~/.layer/bill/config/app.toml

echo "Set pruning to custom..."
sed -i '' 's/^pruning = "default"/pruning = "custom"/g' ~/.layer/bill/config/app.toml
sed -i '' 's/^pruning-keep-recent = "0"/pruning-keep-recent = "1814400"/g' ~/.layer/bill/config/app.toml
sed -i '' 's/^pruning-interval = "0"/pruning-interval = "10"/g' ~/.layer/bill/config/app.toml

echo "Turn on snapshot service for node"
sed -i '' 's/^snapshot-interval = 0/snapshot-interval = 2000/g' ~/.layer/bill/config/app.toml
sed -i '' 's/^snapshot-keep-recent = 2/snapshot-keep-recent = 5/g' ~/.layer/bill/config/app.toml

echo "set chain id in genesis file to layer..."
sed -i '' 's/"chain_id": .*"/"chain_id": '\"layertest-4\"'/g' ~/.layer/bill/config/genesis.json

echo "Updating vote_extensions_enable_height in genesis.json for bill..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"'  ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

# Update signed_blocks_window in genesis.json for luke
echo "Updating signed_blocks_window in genesis.json for bill..."
jq '.app_state.slashing.params.signed_blocks_window = "500"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json
jq '.app_state.globalfee.params.minimum_gas_prices[0].amount = "0.000025000000000000"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

echo "Modifying timeout_commit in config.toml for bill..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/bill/config/config.toml

echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/bill/config/config.toml
echo "Enable unsafe cors"
sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/bill/config/app.toml

update_ports() {
    file=$1
    # Hardcoded replacements for specific lines, incrementing port numbers by 101
    sed -i '' -e 's|address = "tcp://localhost:1317"|address = "tcp://localhost:1418"|g' \
              -e 's|address = "localhost:9090"|address = "localhost:9191"|g' \
              -e 's|node = "tcp://localhost:26657"|node = "tcp://localhost:26758"|g' \
              -e 's|proxy_app = "tcp://127.0.0.1:26658"|proxy_app = "tcp://127.0.0.1:26759"|g' \
              -e 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://127.0.0.1:26758"|g' \
              -e 's|pprof_laddr = "localhost:6060"|pprof_laddr = "localhost:6161"|g' \
              -e 's|laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:26757"|g' "$file"
}

update_ports $NODE2_CONFIG_DIR/app.toml
update_ports $NODE2_CONFIG_DIR/client.toml
update_ports $NODE2_CONFIG_DIR/config.toml



echo "Starting chain for bill..."
#./layerd start --home ~/.layer/bill --key-name bill --api.enable --api.swagger --keyring-backend test

