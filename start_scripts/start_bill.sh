#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
CHAIN_ID="layertest-4"
# Define paths to the node directories
echo "Defining paths..."
NODE1_HOME_DIR="$HOME/.layer/alice"
NODE2_HOME_DIR="$HOME/.layer/bill"
NODE1_CONFIG_DIR=$NODE1_HOME_DIR"/config"
NODE2_CONFIG_DIR=$NODE2_HOME_DIR"/config"
# AMOUNT_IN_LOYA="45000000000loya"
AMOUNT_IN_LOYA="100000000loya"

# Define bill's node home dir to be read by reporter daemon
export LAYERD_NODE_HOME_BILL=$NODE2_HOME_DIR

# Copy the configuration files from node 1 to node 2
echo "Copying configuration files..."
cp $NODE1_CONFIG_DIR/genesis.json $NODE2_CONFIG_DIR/
cp $NODE1_CONFIG_DIR/app.toml $NODE2_CONFIG_DIR/
cp $NODE1_CONFIG_DIR/client.toml $NODE2_CONFIG_DIR/
cp $NODE1_CONFIG_DIR/config.toml $NODE2_CONFIG_DIR/

# add 101 to port numbers and replace them in node 2's configuration files
update_ports() {
    file=$1
    # Hardcoded replacements for specific lines, incrementing port numbers by 101
    sed -i '' -e 's|address = "tcp://localhost:1317"|address = "tcp://localhost:1418"|g' \
              -e 's|address = "localhost:9090"|address = "localhost:9191"|g' \
              -e 's|node = "tcp://localhost:26657"|node = "tcp://localhost:26758"|g' \
              -e 's|proxy_app = "tcp://127.0.0.1:26658"|proxy_app = "tcp://127.0.0.1:26759"|g' \
              -e 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://127.0.0.1:26758"|g' \
              -e 's|pprof_laddr = "localhost:6060"|pprof_laddr = "localhost:6161"|g' \
              -e 's|laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:26757"|g' "$file"
}

# Update ports in the copied configuration files
echo "Updating ports..."
update_ports $NODE2_CONFIG_DIR/app.toml
update_ports $NODE2_CONFIG_DIR/client.toml
update_ports $NODE2_CONFIG_DIR/config.toml

echo "Configuration files copied and ports updated."

# Obtain Node ID of the first node
echo "Obtaining node ID of the first node..."
NODE_ID_1=$(./layerd tendermint show-node-id --home $NODE1_HOME_DIR)
echo "Node ID of the first node: $NODE_ID_1"

# Listen address and port for the first node (adjust if necessary)
LISTEN_ADDR="localhost"
PORT="26656" # Default P2P port, adjust if your setup is different

PEER_ADDR="$NODE_ID_1@$LISTEN_ADDR:$PORT"

# Update seeds and persistent_peers in node 2's config.toml
echo "Updating seeds and persistent_peers in node 2's config.toml..."
sed -i '' "s/seeds = \"\"/seeds = \"$PEER_ADDR\"/" $NODE2_CONFIG_DIR/config.toml
sed -i '' "s/persistent_peers = \"\"/persistent_peers = \"$PEER_ADDR\"/" $NODE2_CONFIG_DIR/config.toml

echo "Seeds/persistent_peers set."

# get bill's validator pubkey
# echo "Getting bill's validator pubkey..."
# BILL_VAL_PUBKEY=$(./layerd tendermint show-validator --home $NODE2_HOME_DIR)
# BILL_VAL_PUBKEY=$(echo "$BILL_VAL_PUBKEY" | jq -r '.key')
# echo "Bill's validator pubkey: $BILL_VAL_PUBKEY"

# # Define the validator.json content
# VALIDATOR_JSON=$(cat <<EOF
# {
#     "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"$BILL_VAL_PUBKEY"},
#     "amount": "$AMOUNT_IN_LOYA",
#     "moniker": "billmoniker",
#     "identity": "optional identity signature (ex. UPort or Keybase)",
#     "website": "validator's (optional) website",
#     "security": "validator's (optional) security contact email",
#     "details": "validator's (optional) details",
#     "commission-rate": "0.1",
#     "commission-max-rate": "0.2",
#     "commission-max-change-rate": "0.01",
#     "min-self-delegation": "1"
# }
# EOF
# )

# # Save the validator.json content to a file
# echo "Creating bill's validator.json..."
# echo "$VALIDATOR_JSON" > $NODE2_HOME_DIR/config/validator.json

# # Stake Bill as a validator
# echo "Staking bill as a validator..."
# ./layerd tx staking create-validator ~/.layer/bill/config/validator.json --from bill --keyring-backend $KEYRING_BACKEND --keyring-dir ~/.layer/bill --chain-id $CHAIN_ID --fees 600loya --yes

# Modify keyring-backend in client.toml for bill
echo "Modifying keyring-backend in client.toml for bill..."
sed -i '' "s/keyring-backend = \"os\"/keyring-backend = \"$KEYRING_BACKEND\"/" ~/.layer/bill/config/client.toml

# Start the second node
echo "Starting the second node..."
#./layerd start --home $NODE2_HOME_DIR --api.enable --keyring-backend $KEYRING_BACKEND --key-name bill 

