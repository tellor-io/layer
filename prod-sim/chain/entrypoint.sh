#!/bin/bash

# Start the validator in the background
/chain/bin/layerd start --home /chain/validator-${VALIDATOR_NUM}/.layer \
  --key-name="validator-${VALIDATOR_NUM}" \
  --api.enable \
  --api.swagger \
  --grpc.enable \
  --grpc-web.enable \
  --grpc.address="0.0.0.0:9090" &

# Wait for validator to be ready
echo "Waiting for validator node to be ready..."
while ! nc -z localhost 26657; do
  sleep 2
done

# Start the reporter
/chain/start-reporter.sh

# Keep container running
wait 