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

echo "Change denom to loya in genesis file..."
sed -i 's/"stake"/"loya"/g' ~/.layer/alice/config/genesis.json
sed -i 's/"stake"/"loya"/g' ~/.layer/config/genesis.json

echo "Change denom to loya in config files..."
sed -i 's/([0-9]+)stake/1loya/g' ~/.layer/alice/config/app.toml
sed -i 's/([0-9]+)stake/1loya/g' ~/.layer/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/alice/config/app.toml
sed -i 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/config/app.toml

echo "Set the keyring backend in client.toml to environment variable..."
sed -i 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/alice/config/client.toml
sed -i 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml

# Add a validator account for alice
echo "Adding validator account for alice..."
./layerd keys add alice --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

echo "set chain id in genesis file to layer..."
sed -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' ~/.layer/alice/config/genesis.json
sed -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' ~/.layer/config/genesis.json

# Update vote_extensions_enable_height in genesis.json for alice
echo "Updating vote_extensions_enable_height in genesis.json for alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json

# Get address/account for alice to use in gentx tx
echo "Get address/account for alice to use in gentx tx"
ALICE=$(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice)
echo "ALICE: $ALICE"

# Create a tx to give alice loyas to stake
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account $ALICE 100000000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

# Create a tx to stake some loyas for alice
echo "Creating gentx for alice..."
./layerd genesis gentx alice 100000000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice --chain-id layer

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice

# validate genesis file
echo "Validate genesis file"
./layerd genesis validate-genesis --home ~/.layer/alice



# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for alice..."
sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/alice/config/config.toml

# Open up alice to outside traffic
echo "Open up alice to outside traffice" 
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/alice/config/config.toml
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/alice/config/config.toml

sed -i 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/alice/config/config.toml



# Modify cors to accept *
echo "Modify cors to accept *"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/alice/config/config.toml
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/config.toml


# enable unsafe cors
echo "Enable unsafe cors"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/alice/config/app.toml
sed -i 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/alice/config/app.toml

sed -i 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml

# set the external address for which to connect to
echo "Setting external address to connect to for aws instance"
sed -i 's/^external_address = ""/external_address = "54.166.101.67:26656"/g' ~/.layer/alice/config/config.toml
sed -i 's/^external_address = ""/external_address = "54.166.101.67:26656"/g' ~/.layer/config/config.toml 

# # Modify keyring-backend in client.toml for alice
# echo "Modifying keyring-backend in client.toml for alice..."
# sed -i 's/^keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/alice/config/client.toml
# # update for main dir as well. why is this needed?
# sed -i 's/keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/config/client.toml

echo "Starting chain for alice..."
./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger