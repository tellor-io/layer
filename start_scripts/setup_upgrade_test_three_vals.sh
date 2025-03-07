#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
#set -e

KEYRING_BACKEND="test"
PASSWORD="password"
export LUKE_NETWORK_ADDRESS="3.80.39.230" #us east
export YODA_NETWORK_ADDRESS="3.101.12.103" #us west
export PALPATINE_NETWORK_ADDRESS="54.255.188.214"

for name in luke yoda palpatine; do
    export LAYERD_NODE_HOME_$name="$HOME/.layer/$name"
done

# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layertest-3

# Initialize chain nodes
for name in luke yoda palpatine; do    
    echo "Initializing chain node for $name..."
    ./layerd init $name-moniker --chain-id layertest-3 --home ~/.layer/$name
    
    echo "Change denom to loya in genesis file..."
    sed -i '' 's/"stake"/"loya"/g' ~/.layer/$name/config/genesis.json

    echo "Change denom to loya in config files for $name..."
    sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/$name/config/app.toml

    echo "Set the keyring backend in client.toml to environment variable for $name..."
    sed -i '' 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/$name/config/client.toml

    echo "Set Chain Id to layer in client config file for $name..."
    sed -i '' 's/^chain-id = .*$/chain-id = "layertest-3"/g' ~/.layer/$name/config/app.toml

    echo "Set pruning to custom..."
    sed -i '' 's/^pruning = "default"/pruning = "custom"/g' ~/.layer/$name/config/app.toml
    sed -i '' 's/^pruning-keep-recent = "0"/pruning-keep-recent = "1209600"/g' ~/.layer/$name/config/app.toml
    sed -i '' 's/^pruning-interval = "0"/pruning-interval = "10"/g' ~/.layer/$name/config/app.toml

    echo "Turn on snapshot service for node"
    sed -i '' 's/^snapshot-interval = 0/snapshot-interval = 2000/g' ~/.layer/$name/config/app.toml
    sed -i '' 's/^snapshot-keep-recent = 2/snapshot-keep-recent = 5/g' ~/.layer/$name/config/app.toml

    echo "set chain id in genesis file to layer..."
    sed -i '' 's/"chain_id": .*"/"chain_id": '\"layertest-3\"'/g' ~/.layer/$name/config/genesis.json

    echo "Updating vote_extensions_enable_height in genesis.json for $name..."
    jq '.consensus.params.abci.vote_extensions_enable_height = "1"'  ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json

    echo "Shorten governance params for testing"
    jq '.gov.params.expedited_voting_period = "600s"'  ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json

    echo "start with minting turned on"
    jq '.mint.initialized = true'  ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json
    sleep 10

    # Update signed_blocks_window in genesis.json for luke
    echo "Updating signed_blocks_window in genesis.json for $name..."
    jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json
    jq '.app_state.globalfee.params.minimum_gas_prices[0].amount = "0.000025000000000000"' ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json

    echo "Modifying timeout_commit in config.toml for $name..."
    sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/$name/config/config.toml

    echo "Open up $name to outside traffic" 
    sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/$name/config/config.toml
    sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/$name/config/config.toml
    sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/$name/config/app.toml
    sed -i '' 's/^address = "localhost:9090"/address = "0.0.0.0:9090"/g' ~/.layer/$name/config/app.toml

    echo "Modify cors to accept *"
    sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$name/config/config.toml
    echo "Enable unsafe cors"
    sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/$name/config/app.toml
done

# Create keys for each account
for name in luke yoda palpatine; do
    echo "Adding validator account for $name..."
    ./layerd keys add $name --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name 2>&1 | tee $name-validator_keys.txt
done

echo "creating account for faucet..."
./layerd keys add faucet --recover=true --keyring-backend test

echo "Get the address for all nodes to use in future steps"
for name in luke yoda palpatine; do
    echo "Get address/account for $name to use in gentx"
    ADDRESS=$(./layerd keys show $name -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name)
    ./layerd genesis add-genesis-account $ADDRESS 200000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/luke
    ./layerd genesis add-genesis-account $ADDRESS 200000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name
done

# Create a tx to give faucet loyas to have on hold to give to users
echo "Adding genesis account for faucet..."
./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 1000000000000loya --home ~/.layer/luke

echo "Add team address to genesis..."
./layerd genesis add-team-account tellor18wjwgr0j8pv4ektdaxvzsykpntdylftwz8ml97 --home ~/.layer/luke

echo "add tokens to team account"
./layerd genesis add-genesis-account tellor18wjwgr0j8pv4ektdaxvzsykpntdylftwz8ml97 1000000000loya --home ~/.layer/luke

for name in luke yoda palpatine; do
    echo "Creating gentx for $name....."
    ADDRESS=$(./layerd keys show $name -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name)
    ./layerd genesis gentx $name 100000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name --chain-id layertest-3
done

for name in yoda palpatine; do
    cp ~/.layer/$name/config/gentx/gentx-* \
        ~/.layer/luke/config/gentx
done 

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/luke

# validate genesis file
echo "Validate genesis file"
./layerd genesis validate-genesis --home ~/.layer/luke

for name in yoda palpatine; do
    cp ~/.layer/luke/config/genesis.json ~/.layer/$name/config/genesis.json
done

echo "Get node id for luke to use for peer identifier"
LUKE_NODE_ID=$(./layerd comet show-node-id --home ~/.layer/luke)
LUKE_NODE_IDENTIFIER=$LUKE_NODE_ID@$LUKE_NETWORK_ADDRESS:26656
echo "luke ip: $LUKE_NODE_IDENTIFIER"

echo "Get node id for yoda to use for peer identifier"
YODA_NODE_ID=$(./layerd comet show-node-id --home ~/.layer/yoda)
echo "yoda node id: $YODA_NODE_ID"
YODA_NODE_IDENTIFIER=$YODA_NODE_ID@$YODA_NETWORK_ADDRESS:26656
echo "yoda ip: $YODA_NODE_IDENTIFIER"

echo "Get node id for palpatine to use for peer identifier"
PALPATINE_NODE_ID=$(./layerd comet show-node-id --home ~/.layer/palpatine)
PALPATINE_NODE_IDENTIFIER=$PALPATINE_NODE_ID@$PALPATINE_NETWORK_ADDRESS:26656
echo "PALPATINE ip: $PALPATINE_NODE_IDENTIFIER"

LUKE_PEERS=$YODA_NODE_IDENTIFIER,$PALPATINE_NODE_IDENTIFIER
YODA_PEERS=$LUKE_NODE_IDENTIFIER,$PALPATINE_NODE_IDENTIFIER
PALPATINE_PEERS=$LUKE_NODE_IDENTIFIER,$YODA_NODE_IDENTIFIER

echo "Set persistent peers"
sed -i '' "s/^persistent_peers = */persistent_peers = \"$LUKE_PEERS\"/g" ~/.layer/luke/config/config.toml
sed -i '' "s/^persistent_peers = \"\"/persistent_peers = \"$YODA_PEERS\"/g" ~/.layer/yoda/config/config.toml
sed -i '' "s/^persistent_peers = \"\"/persistent_peers = \"$PALPATINE_PEERS\"/g" ~/.layer/palpatine/config/config.toml

echo "Luke Peers: $LUKE_PEERS"

# Below is the start command we use when wanting to start the node with the reporter daemon turned on
# ./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false