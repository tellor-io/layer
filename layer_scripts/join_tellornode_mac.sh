#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"

export LAYERD_NODE_HOME="$HOME/.layer/dave"
export NODE_NAME="dave"

# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Initialize chain node with the folder for dave
echo "Initializing chain node for dave..."
./layerd init davemoniker --chain-id layer --home ~/.layer/dave

# Add a validator account for dave
echo "Adding validator account for dave..."
./layerd keys add dave --home ~/.layer/dave --keyring-backend "$KEYRING_BACKEND"

#overwrite genisis.json with actual agreed one
echo "retrieving tellornode genesis.json"
curl tellornode.com:26657/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json
curl tellornode.com:26657/genesis | jq '.result.genesis' > ~/.layer/$NODE_NAME/config/genesis.json

# Modify timeout_commit in config.toml for dave
echo "Modifying timeout_commit in config.toml for dave..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/dave/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffice" 
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/$NODE_NAME/config/config.toml

sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/$NODE_NAME/config/app.toml

# Modify cors to accept *
echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/config.toml

# enable unsafe cors
echo "Enable unsafe cors"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml

#set dave's external_address for connections to other nodes
echo setting ip address for connections
export QUOTED_IP_ADDRESS="$(dig TXT +short o-o.myaddr.l.google.com @ns1.google.com)"
export NODE_IP_ADDRESS=${QUOTED_IP_ADDRESS//\"/}
sed -i '' 's/external_address = ""/external_address = "tcp:\/\/'$NODE_IP_ADDRESS':26656"/g' ~/.layer/$NODE_NAME/config/config.toml

#retrieve tellornode node ID for seeding dave's node
export QUOTED_TELLORNODE_ID="$(curl tellornode.com:26657/status | jq '.result.node_info.id')"
export TELLORNODE_ID=${QUOTED_TELLORNODE_ID//\"/}
sed -i '' 's/seeds = ""/seeds = "'$TELLORNODE_ID'@tellornode.com:26656"/g' ~/.layer/$NODE_NAME/config/config.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for node..."
sed -i '' 's/^keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/$NODE_NAME/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/config/client.toml

./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false
