#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"
NODE_MONIKER="billmoniker"
NODE_NAME="bill"

export LAYERD_NODE_HOME="$HOME/.layer/$NODE_NAME"

# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Initialize chain node with the folder for node
echo "Initializing chain node for node..."
./layerd init $NODE_MONIKER --chain-id layer --home ~/.layer/$NODE_NAME

echo "Change denom to loya in config files..."
sed -i 's/([0-9]+)stake/1loya/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i 's/([0-9]+)stake/1loya/g' ~/.layer/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/config/app.toml

echo "Set the keyring backend in client.toml to environment variable..."
sed -i 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/$NODE_NAME/config/client.toml
sed -i 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml

# # Add a validator account for node
echo "Creating account keys for node to be able to send and receive loya and stake..."
./layerd keys add $NODE_NAME --keyring-backend $KEYRING_BACKEND --home ~/.layer/$NODE_NAME

# Get address/account for node to use in gentx tx
echo "Get address/account for node to use in gentx tx"
NODE=$(./layerd keys show $NODE_NAME -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$NODE_NAME)
echo "NODE: $NODE"

# # Create a tx to give node loyas to stake
# echo "Adding genesis account for node..."
# ./layerd genesis add-genesis-account $NODE 100000000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$NODE_NAME

# # Create a tx to stake some loyas for node
# echo "Creating gentx for node..."
# ./layerd genesis gentx $NODE_NAME 100000000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$NODE_NAME --chain-id layer

# # Add the transactions to the genesis block
# echo "Collecting gentxs..."
# ./layerd genesis collect-gentxs --home ~/.layer/$NODE_NAME

# validate genesis file
# echo "Validate genesis file"
# ./layerd genesis validate-genesis --home ~/.layer/$NODE_NAME



# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for node..."
sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/$NODE_NAME/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffice" 
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/$NODE_NAME/config/config.toml

sed -i 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/$NODE_NAME/config/app.toml



# Modify cors to accept *
echo "Modify cors to accept *"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/config.toml


# enable unsafe cors
echo "Enable unsafe cors"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/$NODE_NAME/config/app.toml

sed -i 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for node..."
sed -i 's/^keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/$NODE_NAME/config/client.toml
# update for main dir as well. why is this needed?
sed -i 's/keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/config/client.toml

# get genesis file from running node's rpc
echo "Getting genesis from runnning node....."
curl tellornode.com:26657/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json
curl tellornode.com:26657/genesis | jq '.result.genesis' > ~/.layer/$NODE_NAME/config/genesis.json

# current running SSL NGINX VERSION node id: 5a47f0cd16e3a6b35a222cbc461b9ab6695cc9d5
# other node id: 0ae46718d9d95ab32bc12f5b4587a9a13dc85ca5
# ALICE_NODE_ID=$(./layerd --node https://tellorlayer.com/rpc comet show-node-id)
# echo "ALICE NODE ID: $ALICE_NODE_ID"
ALICE_NODE_ID=0ae46718d9d95ab32bc12f5b4587a9a13dc85ca5
ALICE_ID=$ALICE_NODE_ID@http://tellornode.com:26656

sed -i '''s/persistent_peers = ""/persistent_peers = "'$ALICE_ID'"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '''s/seeds = ""/seeds = "'$ALICE_ID'"/g' ~/.layer/$NODE_NAME/config/config.toml


echo "Starting chain for node..."
./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false