#!/bin/bash

# Stop execution if any command fails
set -e

# Remove old test chains (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json

# Add a validator account
echo "Adding validator account..."
./layerd keys add alice --keyring-backend test

# Create a tx to give the alice loyas to stake
echo "Adding genesis account..."
./layerd genesis add-genesis-account alice 10000000000000loya

# Create a tx to stake some loyas for alice
echo "Creating gentx..."
./layerd genesis gentx alice 1000000000000loya --chain-id layer

# Add the transactions to the genesis block
echo "Collecting gentxs..."
./layerd genesis collect-gentxs

echo "Start chain..."
./layerd start
