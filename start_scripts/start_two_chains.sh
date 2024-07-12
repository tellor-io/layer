#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"

export LAYERD_NODE_HOME="$HOME/.layer/alice"

# Remove old test chains (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Init two different chain nodes with two different folders
echo "Initializing chain nodes..."
echo "alice..."
./layerd init alicemoniker --chain-id layer --home ~/.layer/alice
echo "bill..."
./layerd init billmoniker --chain-id layer --home ~/.layer/bill

# Add a validator account alice
echo "Adding validator accounts..."
echo "alice..."
./layerd keys add alice --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
echo "bill..."
./layerd keys add bill --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill
echo "charlie..."
yes | ./layerd keys add charlie --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice > ~/Desktop/charlie_key_info.txt 2>&1

# # Extract the mnemonic from the key_info file
# echo "Extracting charlie's mnemonic from key_info file..."
# grep -A 24 'It is the only way to recover your account if you ever forget your password.' ~/Desktop/charlie_key_info.txt | tail -n 1 > ~/Desktop/charlie_mnemonic.txt


# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "main..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
echo "bill..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json


# Update signed_blocks_window in genesis.json for alice
echo "Updating signed_blocks_window in genesis.json for alice..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "Updating signed_blocks_window in genesis.json for bill..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

# Create a tx to give alice loyas to stake
echo "Adding genesis accounts..."
echo "alice..."
./layerd genesis add-genesis-account $(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice)  10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
echo "bill..."
./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
echo "charlie..."
./layerd genesis add-genesis-account $(./layerd keys show charlie -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
# ./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend os --home ~/.layer/bill) 10000000000000loya --keyring-backend os --home ~/.layer/bill

# Create a tx to stake some loyas for alice
echo "Creating gentx alice..."
./layerd genesis gentx alice 1000000000000loya --chain-id layer --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice --keyring-dir ~/.layer/alice

# Add the transactions to the genesis block:q
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice

# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for alice..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "500ms"/' ~/.layer/alice/config/config.toml

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for alice..."
sed -i '' "s/keyring-backend = \"os\"/keyring-backend = \"$KEYRING_BACKEND\"/" ~/.layer/alice/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' "s/keyring-backend = \"os\"/keyring-backend = \"$KEYRING_BACKEND\"/" ~/.layer/config/client.toml


echo "Start chain..."
./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger
