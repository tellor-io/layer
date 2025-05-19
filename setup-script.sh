#!/bin/bash
# setup.sh - Generate validator keys and prepare the test environment

set -e

CHAIN_DIR=$(pwd)/prod-sim/chain
VAL_INFO="${CHAIN_DIR}/validator-info"
GENESIS_DIR="${CHAIN_DIR}/genesis"
BINARY_NAME=./layerd

# Function to create node ID file from node_key.json
extract_node_id() {
    local node_key=$1
    local output_file=$2
    
    # Extract node ID from node_key.json
    NODE_ID=$(${BINARY_NAME} tendermint show-node-id --home $node_key)
    echo $NODE_ID > $output_file
}

echo "Setting up validator keys and genesis file for ${BINARY_NAME} testnet"

# Create directories if they don't exist
mkdir -p ${VAL_INFO}
mkdir -p ${GENESIS_DIR}

MAIN_VALIDATOR_HOME="${CHAIN_DIR}/tmp/validator-0"
# Generate keys for each validator
for i in {0..3}; do
    echo "Setting up validator-${i}..."
    VALIDATOR_HOME="${CHAIN_DIR}/tmp/validator-${i}"
    mkdir -p ${VALIDATOR_HOME}
    
    # Initialize the node to generate keys if they don't exist
    ${BINARY_NAME} init validator-${i} --chain-id tellor-devnet --home ${VALIDATOR_HOME}
    
    # Create validator key directory
    mkdir -p ${VAL_INFO}/validator-${i}
    
    # Copy key files to validator-keys directory
    cp ${VALIDATOR_HOME}/config/priv_validator_key.json ${VAL_INFO}/validator-${i}/
    cp ${VALIDATOR_HOME}/config/node_key.json ${VAL_INFO}/validator-${i}/
    
    # Create node_id file
    extract_node_id ${VALIDATOR_HOME} ${VAL_INFO}/validator-${i}/node_id
    
    # Create validator account
    echo "Creating validator-${i} account..."
    ${BINARY_NAME} keys add validator-${i} --keyring-backend test --home ${VALIDATOR_HOME}
    
    # Fund the account with tokens (adjust this command for your chain's specifics)
    ${BINARY_NAME} genesis add-genesis-account validator-${i} 100000000loya --keyring-backend test --home ${VALIDATOR_HOME}
    
    # Create genesis transaction
    ${BINARY_NAME} genesis gentx validator-${i} 1000000loya --chain-id tellor-devnet --keyring-backend test --home ${VALIDATOR_HOME}

    # add keyring file to val info
    cp -r ${VALIDATOR_HOME}/keyring-test ${VAL_INFO}/validator-${i}/
    
    if [ $i -eq 0 ]; then
        # Use the first validator's genesis as the base
        cp ${VALIDATOR_HOME}/config/genesis.json ${GENESIS_DIR}/
    else
        # Collect gentxs from other validators
        cp ${VALIDATOR_HOME}/config/gentx/* ${CHAIN_DIR}/tmp/validator-0/config/gentx/
        echo "add genesis account to the genesis file of main validator so we can collect gentxs"
        ADDRESS=$(${BINARY_NAME} keys show validator-${i} --keyring-backend test --home ${VALIDATOR_HOME} --output json | jq -r '.address')
        ${BINARY_NAME} genesis add-genesis-account ${ADDRESS} 100000000loya --keyring-backend test --home ${MAIN_VALIDATOR_HOME}
    fi
done

# Go to the first validator directory to collect and finalize genesis
#cd ${CHAIN_DIR}/tmp/validator-0

VALIDATOR_HOME="${CHAIN_DIR}/tmp/validator-0"

# Collect genesis transactions
${BINARY_NAME} genesis collect-gentxs --home ${VALIDATOR_HOME}

# Validate genesis
${BINARY_NAME} genesis validate-genesis --home ${VALIDATOR_HOME}

jq '.consensus.params.abci.vote_extensions_enable_height = "1" | .app_state.slashing.params.signed_blocks_window = "500" | .app_state.gov.params.expedited_voting_period = "300s" | .app_state.gov.params.expedited_min_deposit[0].amount = "10000" | .app_state.gov.params.expedited_min_deposit = [{"denom": "loya", "amount": "10000"}] | .app_state.globalfee.params.minimum_gas_prices[0].amount = "0.000025000000000000" | .app_state.mint.initialized = true' ${VALIDATOR_HOME}/config/genesis.json > ${CHAIN_DIR}/temp.json && mv ${CHAIN_DIR}/temp.json ${VALIDATOR_HOME}/config/genesis.json

# Copy the final genesis to the genesis directory
cp ${VALIDATOR_HOME}/config/genesis.json ${GENESIS_DIR}/

echo "Genesis file created at ${GENESIS_DIR}/genesis.json"
echo "Validator keys created at ${VAL_INFO}"

# Clean up temporary directories
echo "Cleaning up temporary files..."
cd ${CHAIN_DIR}
rm -rf ${CHAIN_DIR}/tmp

echo "Setup complete! You can now run 'docker-compose up' to start the testnet."
