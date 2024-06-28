#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

## YOU WILL NEED TO SET THIS TO WHATEVER NODE YOU WOULD LIKE TO USE
export LAYER_NODE_URL=tellornode.com
export KEYRING_BACKEND="test"
export NODE_MONIKER="billmoniker"
export NODE_NAME="bill"
export TELLORNODE_ID=9eb337547c01106a92ee4727e40ec103d1741a3a
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

# Create a validator account for node
echo "Creating account keys for node to be able to send and receive loya and stake..."
./layerd keys add $NODE_NAME --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME

# Import validator account from seed phrase
# echo "Importing validator account from seed phrase..."
# ./layerd keys add $NODE_NAME --recover=true --keyring-backend $KEYRING_BACKEND

# Get address/account for node to use in gentx tx
echo "Getting the address of your node to use for faucet request"
NODE_ADDRESS=$(./layerd keys show $NODE_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME)
echo "NODE address: $NODE_ADDRESS"
sleep 10

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
sed -i 's/^keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/$NODE_NAME/config/client.toml
# update for main dir as well. why is this needed?
sed -i 's/keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml

rm -f ~/.layer/config/genesis.json
rm -f ~/.layer/$NODE_NAME/config/genesis.json
# get genesis file from running node's rpc
echo "Getting genesis from runnning node....."
curl $LAYER_NODE_URL:26657/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json
curl $LAYER_NODE_URL:26657/genesis | jq '.result.genesis' > ~/.layer/$NODE_NAME/config/genesis.json

sed -i 's/seeds = ""/seeds = "'$TELLORNODE_ID'@'$LAYER_NODE_URL':26656"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i 's/persistent_peers = ""/persistent_peers = "'$TELLORNODE_ID'@'$LAYER_NODE_URL':26656"/g' ~/.layer/$NODE_NAME/config/config.toml

echo "Path: $TELLORNODE_ID@$LAYER_NODE_URL:26656"

echo "Starting chain for node..."

#./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false --p2p.seeds "$TELLORNODE_ID@$LAYER_NODE_URL:26656"
./layerd start --home $LAYERD_NODE_HOME --api.swagger --price-daemon-enabled=false --p2p.seeds "$TELLORNODE_ID@$LAYER_NODE_URL:26656" | tee ./second_node_logs.txt
#./layerd start --home ~/.layer/bill --api.enable --api.swagger --panic-on-daemon-failure-enabled=false --p2p.seeds "9eb337547c01106a92ee4727e40ec103d1741a3a@tellornode.com:26656" | tee ./second_node_logs.txt


# ec2-54-166-101-67.compute-1.amazonaws.com
# sudo scp -i /Users/caleb/layer-doc-test-key.pem ubuntu@ec2-100-26-53-93.compute-1.amazonaws.com:/home/ubuntu/layer/second_node_logs.txt .
# sudo scp -i /Users/caleb/layer-testnet.pem ubuntu@ec2-54-166-101-67.compute-1.amazonaws.com:/home/ubuntu/layer/first_node_logs.txt .

# // "slashing": {
#       "params": {
#         "signed_blocks_window": "100",
#         "min_signed_per_window": "0.500000000000000000",
#         "downtime_jail_duration": "600s",
#         "slash_fraction_double_sign": "0.050000000000000000",
#         "slash_fraction_downtime": "0.010000000000000000"
#       },
#       "signing_infos": [],
#       "missed_blocks": []
#     },