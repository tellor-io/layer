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

echo "creating account for faucet..."
./layerd keys add faucet --recover=true --keyring-backend test

# Create account for second validator
echo "Adding account for Bill..."
echo "Make sure to save this seed phrase and address as you will need them in the future when creating their node..."
./layerd keys add bill --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill
sleep 10

# Create account for third validator
echo "Adding account for Bob..."
echo "Make sure to save this seed phrase and address as you will need them in the future when creating their node..."
./layerd keys add bob --keyring-backend $KEYRING_BACKEND --home ~/.layer/bob
sleep 10

# Create account for Fourth validator
echo "Adding account for Tom..."
echo "Make sure to save this seed phrase and address as you will need them in the future when creating their node..."
./layerd keys add tom --keyring-backend $KEYRING_BACKEND --home ~/.layer/tom
sleep 10

echo "set chain id in genesis file to layer..."
sed -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' ~/.layer/alice/config/genesis.json
sed -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' ~/.layer/config/genesis.json

# Update vote_extensions_enable_height in genesis.json for alice
echo "Updating vote_extensions_enable_height in genesis.json for alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"  ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
jq '.consensus.params.abci.vote_extensions_enable_height = "1"  ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json

# Update signed_blocks_window in genesis.json for alice
echo "Updating signed_blocks_window in genesis.json for alice..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/alice/config/genesis.json > temp.json && mv temp.json ~/.layer/alice/config/genesis.json
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json

# Get address/account for alice to use in gentx tx
echo "Get address/account for alice to use in gentx tx"
ALICE=$(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice)
echo "ALICE: $ALICE"

echo "Get address/account for bill to use in gentx tx"
BILL=$(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill)
echo "BILL: $BILL"

echo "Get address/account for bob to use in gentx tx"
BOB=$(./layerd keys show bob -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bob)
echo "Bob: $BOB"

echo "Get address/account for Tom to use in gentx tx"
TOM=$(./layerd keys show tom -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/tom)
echo "Bob: $TOM"

# echo "Get address for faucet account..."
# FAUCET=$(./layerd keys show faucet -a )
# echo "Faucet keys: $FAUCET"

# Create a tx to give alice loyas to stake
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account $ALICE 100000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

echo "Initializing Bill account with loya to stake.."
./layerd genesis add-genesis-account $BILL 100000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

echo "Initializing Bob account with loya to stake.."
./layerd genesis add-genesis-account $BOB 100000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

echo "Initializing Tom account with loya to stake.."
./layerd genesis add-genesis-account $TOM 100000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice

# Create a tx to give faucet loyas to have on hold to give to users
echo "Adding genesis account for alice..."
./layerd genesis add-genesis-account tellor19d90wqftqx34khmln36zjdswm9p2aqawq2t3vp 1000000000000000000000000000loya --home ~/.layer/alice

# Create a tx to stake some loyas for alice
echo "Creating gentx for alice..."
./layerd genesis gentx alice 1000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/alice --chain-id layer

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

sed -i 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/alice/config/app.toml
sed -i 's/^address = "localhost:9090"/address = "0.0.0.0:9090"/g' ~/.layer/alice/config/app.toml


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

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for alice..."
sed -i 's/^keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/alice/config/client.toml
# update for main dir as well. why is this needed?
sed -i 's/keyring-backend = "os"/keyring-backend = "test"/g' ~/.layer/config/client.toml

echo "Starting chain for alice..."
./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false