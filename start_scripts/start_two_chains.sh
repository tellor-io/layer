#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"
KEY_NAME="alice"
CHAIN_ID="layertest-4"


export LAYERD_NODE_HOME_1="$HOME/.layer/$KEY_NAME"

# Remove old test chains (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init $CHAIN_ID --chain-id $CHAIN_ID

# Init two different chain nodes with two different folders
echo "Initializing chain nodes..."
echo "$KEY_NAME..."
./layerd init alicemoniker --chain-id $CHAIN_ID --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd init billmoniker --chain-id $CHAIN_ID --home ~/.layer/bill

# Add a validator account alice
echo "Adding validator accounts..."
echo "$KEY_NAME..."
./layerd keys add $KEY_NAME --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd keys add bill --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill
echo "charlie..."
#yes | ./layerd keys add charlie --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME > ~/Desktop/charlie_key_info.txt 2>&1

# # Extract the mnemonic from the key_info file
# echo "Extracting charlie's mnemonic from key_info file..."
# grep -A 24 'It is the only way to recover your account if you ever forget your password.' ~/Desktop/charlie_key_info.txt | tail -n 1 > ~/Desktop/charlie_mnemonic.txt


# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "main..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "$KEY_NAME..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
echo "bill..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json


# Update signed_blocks_window in genesis.json for alice
echo "Updating signed_blocks_window in genesis.json for $KEY_NAME..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "Updating signed_blocks_window in genesis.json for bill..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

# Create a tx to give alice loyas to stake
echo "Adding genesis accounts..."
echo "$KEY_NAME..."
./layerd genesis add-genesis-account $(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1)  10000000000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill
./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice


#echo "charlie..."
#./layerd genesis add-genesis-account $(./layerd keys show charlie -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME
# ./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend os --home ~/.layer/bill) 10000000000000loya --keyring-backend os --home ~/.layer/bill

# Create a tx to stake some loyas for alice
echo "Creating gentx $KEY_NAME..."
./layerd genesis gentx $KEY_NAME 1000000000loya --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1 --keyring-dir $LAYERD_NODE_HOME_1

echo "creating gentx for bill"
./layerd genesis gentx bill 1000000000loya --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill --keyring-dir ~/.layer/bill

cp ~/.layer/bill/config/gentx/* ~/.layer/alice/config/gentx/

# Add the transactions to the genesis block:q
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home $LAYERD_NODE_HOME_1

./layerd genesis validate-genesis --home $LAYERD_NODE_HOME_1

cp ~/.layer/alice/config/genesis.json ~/.layer/bill/config/genesis.json


# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for $KEY_NAME..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYERD_NODE_HOME_1/config/config.toml
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/bill/config/config.toml

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for $KEY_NAME..."
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" $LAYERD_NODE_HOME_1/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" ~/.layer/config/client.toml


echo "Start chain..."
#./layerd start --home $LAYERD_NODE_HOME_1 --api.enable --api.swagger --keyring-backend $KEYRING_BACKEND --key-name $KEY_NAME
