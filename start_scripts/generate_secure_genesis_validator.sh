#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

export KEYRING_BACKEND="test"
export PASSWORD="password"
export MONIKER="lukemoniker"
export KEY_NAME="luke"
export CHAIN_ID="layertest-4"
export LAYERD_HOME="~/.layer/luke"

# Remove old test chain data (if present)
echo "Removing old test chain data..."
sudo rm -rf $LAYERD_HOME

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize chain node with the folder for validator
echo "Initializing chain node for $KEY_NAME..."
./layerd init $MONIKER --chain-id $CHAIN_ID --home $LAYERD_HOME

echo "Change denom to loya in genesis file..."
sed -i 's/"stake"/"loya"/g' $LAYERD_HOME/config/genesis.json

echo "Change denom to loya in config files..."
sed -i 's/([0-9]+)stake/1loya/g' $LAYERD_HOME/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i 's/^chain-id = .*$/chain-id = '\"$CHAIN_ID\"'/g' $LAYERD_HOME/config/app.toml

echo "Set the keyring backend in client.toml to environment variable..."
sed -i 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' $LAYERD_HOME/config/client.toml

# Add a validator account for validator
echo "Adding validator account for $KEY_NAME..."
./layerd keys add $KEY_NAME --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME

echo "set chain id in genesis file to layer..."
sed -i 's/"chain_id": .*"/"chain_id": '\"$CHAIN_ID\"'/g' $LAYERD_HOME/config/genesis.json

# Update vote_extensions_enable_height in genesis.json for node
echo "Updating vote_extensions_enable_height in genesis.json for $KEY_NAME..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1" | .slashing.params.signed_blocks_window = "500"' $LAYERD_HOME/config/genesis.json > temp.json && mv temp.json $LAYERD_HOME/config/genesis.json
jq '.app_state.globalfee.params.minimum_gas_prices[0].amount = "0.000025000000000000"' $LAYERD_HOME/config/genesis.json > temp.json && mv temp.json $LAYERD_HOME/config/genesis.json

echo "Set pruning to custom..."
sed -i '' 's/^pruning = "default"/pruning = "custom"/g' $LAYERD_HOME/config/app.toml
sed -i '' 's/^pruning-keep-recent = "0"/pruning-keep-recent = "1209600"/g' $LAYERD_HOME/config/app.toml
sed -i '' 's/^pruning-interval = "0"/pruning-interval = "10"/g' $LAYERD_HOME/config/app.toml

echo "Turn on snapshot service for node"
sed -i '' 's/^snapshot-interval = 0/snapshot-interval = 2000/g' $LAYERD_HOME/config/app.toml
sed -i '' 's/^snapshot-keep-recent = 2/snapshot-keep-recent = 5/g' $LAYERD_HOME/config/app.toml

# Get address/account for validator to use in gentx tx
echo "Get address/account for $KEY_NAME to use in gentx tx"
VAL=$(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME)
echo "VAL: $VAL"

# Create genesis account to give validator loyas to stake
echo "Adding genesis account for $KEY_NAME..."
./layerd genesis add-genesis-account $VAL 100000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME

# Create a tx to stake some loyas for validator
echo "Creating gentx for $KEY_NAME..."
./layerd genesis gentx $KEY_NAME 10000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME --chain-id $CHAIN_ID

echo "Add team address to genesis..."
./layerd genesis add-team-account tellor18wjwgr0j8pv4ektdaxvzsykpntdylftwz8ml97 --home $LAYERD_HOME

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home $LAYERD_HOME

# validate genesis file
echo "Validate genesis file"
./layerd genesis validate-genesis --home $LAYERD_HOME

# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for $KEY_NAME..."
sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYERD_HOME/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffice" 
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' $LAYERD_HOME/config/config.toml
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' $LAYERD_HOME/config/config.toml

sed -i 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' $LAYERD_HOME/config/app.toml
sed -i 's/^address = "localhost:9090"/address = "0.0.0.0:9090"/g' $LAYERD_HOME/config/app.toml

# enable unsafe cors
echo "Enable unsafe cors"
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $LAYERD_HOME/config/app.toml
sed -i 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $LAYERD_HOME/config/app.toml
sed -i 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' $LAYERD_HOME/config/app.toml

echo "Enabled metrics sinks/promethues"
sed -i 's/^enabled = false/enabled = true/g' $LAYERD_HOME/config/app.toml
# echo "Set prometheus retention time to 60s"
sed -i 's/^prometheus-retention-time = 0/prometheus-retention-time = 60/g' $LAYERD_HOME/config/app.toml

echo "Turn on cometBFT prometheus"
sed -i 's/prometheus = false/prometheus = true/' $LAYERD_HOME/config/config.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for $KEY_NAME..."
sed -i 's/keyring-backend = "os"/keyring-backend = "test"/g' $LAYERD_HOME/config/client.toml

#./layerd start --home ~/.layer --key-name="$KEY_NAME" --api.enable --api.swagger
#./layerd start --home ~/.layer/alice --key-name alice --api.enable --api.swagger | tee ./fulldata_first_node_logs.txt | grep 'failed to execute message' >> "filtered_first_node_logs.txt"
