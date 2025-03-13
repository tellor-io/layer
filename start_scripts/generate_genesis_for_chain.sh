#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

export DELEGATOR_ONE_ADD=""

export VALIDATOR_TWO_ACC_ADD=""
export DELEGATOR_TWO_ADD=""

export VALIDATOR_THREE_ACC_ADD=""
export DELEGATOR_THREE_ADD=""

export VALIDATOR_FOUR_ACC_ADD=""
export DELEGATOR_FOUR_ADD=""

export VALIDATOR_FIVE_ACC_ADD=""
export DELEGATOR_FIVE_ADD=""

export VALIDATOR_SIX_ACC_ADD=""
export DELEGATOR_SIX_ADD=""

# create other 5 genesis validator accounts
echo "Create 5 other validator accounts and 6 delegator accounts"
./layerd genesis add-genesis-account $VALIDATOR_TWO_ACC_ADD 100000000loya --home ~/.layer
./layerd genesis add-genesis-account $VALIDATOR_THREE_ACC_ADD 100000000loya --home ~/.layer
./layerd genesis add-genesis-account $VALIDATOR_FOUR_ACC_ADD 100000000loya --home ~/.layer
./layerd genesis add-genesis-account $VALIDATOR_FIVE_ACC_ADD 100000000loya --home ~/.layer
./layerd genesis add-genesis-account $VALIDATOR_SIX_ACC_ADD 100000000loya --home ~/.layer

# Create all 6 genesis delegator accounts
./layerd genesis add-genesis-account $DELEGATOR_ONE_ADD 1000000000loya --home ~/.layer
./layerd genesis add-genesis-account $DELEGATOR_TWO_ADD 1000000000loya --home ~/.layer
./layerd genesis add-genesis-account $DELEGATOR_THREE_ADD 1000000000loya --home ~/.layer
./layerd genesis add-genesis-account $DELEGATOR_FOUR_ADD 1000000000loya --home ~/.layer
./layerd genesis add-genesis-account $DELEGATOR_FIVE_ADD 1000000000loya --home ~/.layer
./layerd genesis add-genesis-account $DELEGATOR_SIX_ADD 1000000000loya --home ~/.layer

echo "Collect genesis tx's"
./layerd genesis collect-gentxs --home ~/.layer

echo "validate genesis file"
./layerd genesis validate-genesis --home ~/.layer
