#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"
NODE_MONIKER="billmoniker"
NODE_NAME="bill"
AMOUNT_IN_TRB=1000
AMOUNT_IN_LOYA=1000000000loya

export LAYERD_NODE_HOME="$HOME/.layer/$NODE_NAME"
## YOU WILL NEED TO SET THIS TO WHATEVER NODE YOU WOULD LIKE TO USE
export LAYER_NODE_URL=tellornode.com

echo "Getting the address of your node to use for faucet request"
NODE_ADDRESS=$(./layerd keys show $NODE_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME)

echo "Calling faucet to fund account..."
curl -X POST localhost:3000/faucetRequest/user/$NODE_ADDRESS/amount/$AMOUNT_IN_TRB

sleep 10

echo "Making a tx to create validator..."
./layerd tx staking create-validator \
  --amount=$AMOUNT_IN_LOYA \
  --pubkey=$(./layerd comet show-validator --home $LAYERD_NODE_HOME) \
  --moniker=$NODE_MONIKER \
  --chain-id=layer \
  --gas="auto" \
  --gas-prices="0.0025loya" \
  --from=$NODE_NAME \
  --node=$LAYER_NODE_URL:26657