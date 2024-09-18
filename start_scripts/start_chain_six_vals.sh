#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
#set -e

KEYRING_BACKEND="test"
PASSWORD="password"
export ALICE_IP_ADDRESS=""
export BILL_IP_ADDRESS=""

for name in alice bill; do
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
./layerd init layer --chain-id layertest-2

# Initialize chain node with the folder for alice
for name in alice bill; do
    echo "Initializing chain node for $name..."
    ./layerd init $name-moniker --chain-id layertest-2 --home ~/.layer/$name
    echo "Change denom to loya in genesis file..."
    sed -i '' 's/"stake"/"loya"/g' ~/.layer/$name/config/genesis.json

    echo "Change denom to loya in config files for $name..."
    sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/$name/config/app.toml

    echo "Set the keyring backend in client.toml to environment variable for $name..."
    sed -i '' 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/$name/config/client.toml

    echo "Set Chain Id to layer in client config file for $name..."
    sed -i '' 's/^chain-id = .*$/chain-id = "layertest-2"/g' ~/.layer/$name/config/app.toml

    echo "set chain id in genesis file to layer..."
    sed -i '' 's/"chain_id": .*"/"chain_id": '\"layertest-2\"'/g' ~/.layer/$name/config/genesis.json

    echo "Updating vote_extensions_enable_height in genesis.json for $name..."
    jq '.consensus.params.abci.vote_extensions_enable_height = "1"'  ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json

    # Update signed_blocks_window in genesis.json for alice
    echo "Updating signed_blocks_window in genesis.json for $name..."
    jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/$name/config/genesis.json > temp.json && mv temp.json ~/.layer/$name/config/genesis.json

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
for name in alice bill; do
    echo "Adding validator account for $name..."
    ./layerd keys add $name --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name 2>&1 | tee $name-validator_keys.txt
done

echo "creating account for faucet..."
./layerd keys add faucet --recover=true --keyring-backend test

echo "Get the address for all nodes to use in future steps"
for name in alice bill; do
    echo "Get address/account for $name to use in gentx"
    ADDRESS=$(./layerd keys show $name -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name)
    ./layerd genesis add-genesis-account $ADDRESS 200000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice
    ./layerd genesis add-genesis-account $ADDRESS 200000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name
done

# Create a tx to give faucet loyas to have on hold to give to users
echo "Adding genesis account for faucet..."
./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 1000000000000000000000000000loya --home ~/.layer/alice

for name in alice bill; do
    echo "Creating gentx for $name....."
    ADDRESS=$(./layerd keys show $name -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name)
    ./layerd genesis gentx $name 1000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$name --chain-id layertest-2
done

cp ~/.layer/bill/config/gentx/gentx-* \
    ~/.layer/alice/config/gentx

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home ~/.layer/alice

# validate genesis file
echo "Validate genesis file"
./layerd genesis validate-genesis --home ~/.layer/alice

for name in bill; do
    cp ~/.layer/alice/config/genesis.json ~/.layer/$name/config/genesis.json
done

echo "Get node id for Alice to use for peer identifier"
ALICE_NODE_ID=$(./layerd comet show-node-id --home ~/.layer/alice)
ALICE_NODE_IDENTIFIER=$NODE_ID@$ALICE_IP_ADDRESS:26656

echo "Get node id for bill to use for peer identifier"
BILL_NODE_ID=$(./layerd comet show-node-id --home ~/.layer/bill)
BILL_NODE_IDENTIFIER=$NODE_ID@$BILL_IP_ADDRESS:26656

ALICE_PEERS=$BILL_NODE_IDENTIFIER
BILL_PEERS=$ALICE_NODE_IDENTIFIER

echo "Set persistent peers in Alice"
sed -i '' "s/^persistent_peers = \"\"/persistent_peers = \"$ALICE_PEERS\"/g" ~/.layer/alice/config/config.toml
sed -i '' "s/^persistent_peers = \"\"/persistent_peers = \"$BILL_PEERS\"/g" ~/.layer/bill/config/config.toml


#echo "Starting chain for alice..."
# ./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false