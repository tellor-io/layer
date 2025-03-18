#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

export KEYRING_BACKEND="test"
export PASSWORD="password"
export KEY_NAME="luke-del"
export CHAIN_ID="layertest-4"
export LAYERD_HOME="~/.layer/luke-del"
export VAL_ADDR=""

echo "Create delegator keys"
./layerd keys add $KEY_NAME --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME

DEL_ADDR=$(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_HOME)

echo "Create delegate transaction message for $KEY_NAME"
./layerd tx staking delegate $VAL_ADDR 1000000000loya --from $DEL_ADDR --chain-id $CHAIN_ID --fees 20loya --generate-only > $LAYERD_HOME/tx_test.json 

echo "Sign the generated delegate transaction"
./layerd tx sign $LAYERD_HOME/tx_test.json --from $DEL_ADDR --chain-id $CHAIN_ID --output-document=$LAYERD_HOME/$DEL_ADDR/signed_delegate_tx.json --keyring-backend test --keyring-dir $LAYERD_HOME/$KEY_NAME --offline --sequence 0 --account-number 0