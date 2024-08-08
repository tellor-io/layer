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
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Initialize chain node with the folder for alice
echo "Initializing chain node for alice..."
./layerd init alicemoniker --chain-id layer --home ~/.layer/alice

# Add a validator account for alice
echo "Adding validator account for alice..."
./layerd keys add alice --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

echo "creating account for faucet..."
./layerd keys add faucet --recover=true --keyring-backend test

# Update vote_extensions_enable_height in genesis.json for alice
echo "Updating vote_extensions_enable_height in genesis.json for alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json

# update the block grace period you have to join as a validator
echo "Updating block grace period from 100 to 500"
jq '.slashing.params.signed_blocks_window = "500"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
jq '.slashing.params.signed_blocks_window = "500"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
sleep 10

# Create a tx to give alice loyas to stake
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account $(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
echo "charlie..."
echo "Adding genesis account for faucet..."
./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 1000000000000000000000000000loya --home ~/.layer/alice

# echo "Faucet..."
# ./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 10000000000000loya --home ~/.layer/alice

# Create a tx to stake some loyas for alice
echo "Creating gentx for alice..."
./layerd genesis gentx alice 1000000000loya --chain-id layer --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice --keyring-dir ~/.layer/alice

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice

# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for alice..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/alice/config/config.toml

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for alice..."
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/alice/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/config/client.toml


# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for alice..."
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/alice/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/config/client.toml


echo "Starting chain for alice..."
./layerd start --home ~/.layer/alice --key-name alice --api.enable --api.swagger | tee ./fulldata_first_node_logs.txt
