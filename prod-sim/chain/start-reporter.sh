#!/bin/bash

# Get validator number from environment
VALIDATOR_NUM="${VALIDATOR_NUM:-0}"
IS_FORK="${IS_FORK:-false}"

echo "Starting reporter script for validator ${VALIDATOR_NUM} (IS_FORK: ${IS_FORK})"

echo "Waiting for validator node to be ready..."
while ! nc -z localhost 26657; do
  sleep 2
done

# Sleep longer if this is a fork
if [ "$IS_FORK" = "true" ]; then
  echo "This is a fork, sleeping for 600 seconds..."
  sleep 600
else
  echo "Sleeping for 10 seconds before making reporter..."
  sleep 10
fi

echo "Validator node is ready"

# Only create reporter if this is not a fork
if [ "$IS_FORK" != "true" ]; then
  echo "Creating reporter..."
  /chain/bin/layerd tx reporter create-reporter 0.1 1000000 reporter-${VALIDATOR_NUM} \
    --from validator-${VALIDATOR_NUM} \
    --chain-id tellor-devnet \
    --fees 20loya \
    --keyring-dir /chain/validator-${VALIDATOR_NUM}/.layer \
    --keyring-backend test \
    --node tcp://localhost:26657 \
    --yes
fi

echo "Starting reporter..."
/chain/bin/reporterd \
  --chain-id tellor-devnet \
  --grpc-addr localhost:9090 \
  --from validator-${VALIDATOR_NUM} \
  --home /chain/validator-${VALIDATOR_NUM}/.layer \
  --keyring-backend test \
  --node tcp://localhost:26657 \
  --broadcast-mode sync \
  --prometheus-port 26661