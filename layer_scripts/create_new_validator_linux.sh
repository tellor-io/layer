#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

### YOU MUST HAVE THE FAUCET RUNNING LOCALLY FOR THIS SCRIPT TO WORK

## YOU WILL NEED TO SET THIS TO WHATEVER NODE YOU WOULD LIKE TO USE
export LAYER_NODE_URL=18.212.102.176
export TELLORNODE_ID=d2ab6de0613631c6f6d6cca3c9bc76309a6ed04d
export KEYRING_BACKEND="test"
export NODE_MONIKER="billmoniker"
export NODE_NAME="bill"
export AMOUNT_IN_TRB=30000
export AMOUNT_IN_LOYA="30000000000loya"
export LAYERD_NODE_HOME="$HOME/.layer/$NODE_NAME"

# # Create a validator account for node
# echo "Creating account keys for node to be able to send and receive loya and stake..."
# ./layerd keys add $NODE_NAME --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME

# Import validator account from seed phrase
# echo "Importing validator account from seed phrase..."
# ./layerd keys add $NODE_NAME --recover=true --keyring-backend $KEYRING_BACKEND

# Get address/account for node to use in gentx tx
echo "Getting the address of your node to use for faucet request"
NODE_ADDRESS=$(./layerd keys show $NODE_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME)
echo "NODE address: $NODE_ADDRESS"

# echo "Calling faucet to fund account..."
# curl -X POST localhost:3000/faucetRequest/user/$NODE_ADDRESS/amount/$AMOUNT_IN_TRB
# sleep 20

VAL_PUB_KEY=$(./layerd comet show-validator --home $LAYERD_NODE_HOME)
echo "Validator's pubkey: $VAL_PUB_KEY"

VALIDATOR_JSON=$(cat <<EOF
{
    "pubkey": $VAL_PUB_KEY,
    "amount": "$AMOUNT_IN_LOYA",
    "moniker": "$NODE_MONIKER",
    "identity": "optional identity signature (ex. UPort or Keybase)",
    "website": "validator's (optional) website",
    "security": "validator's (optional) security contact email",
    "details": "validator's (optional) details",
    "commission-rate": "0.1",
    "commission-max-rate": "0.2",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
}
EOF
)

# Save the validator.json content to a file
echo "Creating bill's validator.json..."
echo "$VALIDATOR_JSON" > ./validator.json

echo "Creating and broadcasting transaction to create validator on chain...."
./layerd tx staking create-validator ./validator.json --from $NODE_ADDRESS --home $LAYERD_NODE_HOME --chain-id layer --node="http://$LAYER_NODE_URL:26657" --gas "400000"

echo "Wait for 2 seconds to allow for validator to be bonded before we query the validator info"
sleep 2

echo "Querying new validator info... Looking for the status field to have a value of 3 which shows that the new validator is bonded"
./layerd query staking validator $(./layerd keys show $NODE_NAME --bech val --address --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME) --output json | jq


echo "If status is 3 now is the time to go back to the screen session or terminal your node is running on and use CTL-C to stop the node"
echo "We will wait in this script for 10 seconds before the chain is restarted."
sleep 10

./layerd start --home $LAYERD_NODE_HOME --key-name $NODE_NAME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false --p2p.seeds "$TELLORNODE_ID@$LAYER_NODE_URL:26656"
