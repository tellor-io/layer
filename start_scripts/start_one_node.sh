#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"

export LAYERD_NODE_HOME="$HOME/.layer/alice"

# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer/alice
rm -rf ~/.layer/config
rm -rf ~/.layer/data

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layertest-2

# Initialize chain node with the folder for alice
echo "Initializing chain node for alice..."
./layerd init alicemoniker --chain-id layertest-2 --home ~/.layer/alice

# Add a validator account for alice
echo "Adding validator account for alice..."
./layerd keys add alice --keyring-backend test --home ~/.layer/alice

echo "creating account for faucet..."
./layerd keys add faucet --recover=true --keyring-backend test

echo "Change denom to loya in genesis file..."
sed -i '' 's/"stake"/"loya"/g' ~/.layer/alice/config/genesis.json

echo "Change denom to loya in config files for alice..."
sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/alice/config/app.toml

echo "Set the keyring backend in client.toml to environment variable for alice..."
sed -i '' 's/^keyring-backend = .*"/keyring-backend = "test"/g' ~/.layer/alice/config/client.toml

echo "Set Chain Id to layer in client config file for alice..."
sed -i '' 's/^chain-id = .*$/chain-id = "layertest-2"/g' ~/.layer/alice/config/app.toml

echo "Set pruning to custom..."
sed -i '' 's/^pruning = "default"/pruning = "custom"/g' ~/.layer/alice/config/app.toml
sed -i '' 's/^pruning-keep-recent = "0"/pruning-keep-recent = "1814400"/g' ~/.layer/alice/config/app.toml
sed -i '' 's/^pruning-interval = "0"/pruning-interval = "10"/g' ~/.layer/alice/config/app.toml

echo "Turn on snapshot service for node"
sed -i '' 's/^snapshot-interval = 0/snapshot-interval = 2000/g' ~/.layer/alice/config/app.toml
sed -i '' 's/^snapshot-keep-recent = 2/snapshot-keep-recent = 5/g' ~/.layer/alice/config/app.toml

echo "set chain id in genesis file to layer..."
sed -i '' 's/"chain_id": .*"/"chain_id": '\"layertest-2\"'/g' ~/.layer/alice/config/genesis.json

echo "Updating vote_extensions_enable_height in genesis.json for alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"'  ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json

# Update signed_blocks_window in genesis.json for luke
echo "Updating signed_blocks_window in genesis.json for alice..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
jq '.app_state.globalfee.params.minimum_gas_prices[0].amount = "0.000025000000000000"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json

echo "Modifying timeout_commit in config.toml for alice..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/alice/config/config.toml

echo "Open up alice to outside traffic" 
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/alice/config/config.toml
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/alice/config/config.toml
sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/alice/config/app.toml
sed -i '' 's/^address = "localhost:9090"/address = "0.0.0.0:9090"/g' ~/.layer/alice/config/app.toml

echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/alice/config/config.toml
echo "Enable unsafe cors"
sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/alice/config/app.toml

# Create a tx to give alice loyas to stake
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account $(./layerd keys show alice -a --keyring-backend test --home ~/.layer/alice) 10000000000000loya --keyring-backend test --home ~/.layer/alice
echo "charlie..."
echo "Adding genesis account for faucet..."
./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 1000000000000000000000000000loya --home ~/.layer/alice

echo "Add team address to genesis..."
./layerd genesis add-team-account tellor18wjwgr0j8pv4ektdaxvzsykpntdylftwz8ml97 --home ~/.layer/alice

# echo "Faucet..."
# ./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 10000000000000loya --home ~/.layer/alice

# Create a tx to stake some loyas for alice
echo "Creating gentx for alice..."
./layerd genesis gentx alice 1000000000loya --chain-id layertest-2 --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice --keyring-dir ~/.layer/alice

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice



echo "Starting chain for alice..."
./layerd start --home ~/.layer/alice --key-name alice --api.enable --api.swagger --keyring-backend test