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

# Initialize chain node with the folder for alice
echo "Initializing chain node for alice..."
./layerd init alicemoniker --chain-id layer --home  ~/.layer/alice

# Add a validator account for alice
echo "Adding validator account for alice..."
./layerd keys add alice --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice
./layerd keys add bill --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice
./layerd keys add team --multisig alice,bill --multisig-threshold 2 --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice

# Update vote_extensions_enable_height in genesis.json for alice
echo "Updating vote_extensions_enable_height in genesis.json for alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json

# Create a tx to give alice loyas to stake
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account $(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice) 10000000000000loya --keyring-backend $KEYRING_BACKEND  --home ~/.layer/alice
echo "Adding team multisig account to dispute params..."
./layerd genesis add-team-account $(./layerd keys show team -a --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice) --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice
# Create a tx to stake some loyas for alice
echo "Creating gentx for alice..."
./layerd genesis gentx alice 1000000000000loya --chain-id layer --keyring-backend $KEYRING_BACKEND --home  ~/.layer/alice --keyring-dir ~/.layer/alice

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice

# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for alice..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "500ms"/' ~/.layer/alice/config/config.toml


# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for alice..."
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/alice/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/' ~/.layer/config/client.toml


echo "Starting chain for alice..."
./layerd start --home  ~/.layer/alice --api.enable --api.swagger