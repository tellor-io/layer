#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"
KEY_NAME="alice"
CHAIN_ID="layertest-4"
STAKE_AMOUNT_1="1000000000000loya"
PRIVATE_KEY_1="60d7de76caa85724ec588a5cd88a2afb77729658bcb783938109d2162779b225" # alice
PRIVATE_KEY_2="e40bf75172f36cb722a6db1042999f7e7b78b92b0d181cdd3a2dfb323595304a" # bill
PRIVATE_KEY_3="a6f6364de568b6a3ecfbf7d494852fc8114d52ef9a94faac987858efec3b1124" # charlie


export LAYERD_NODE_HOME_1="$HOME/.layer/$KEY_NAME"

# Remove old test chains (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init $CHAIN_ID --chain-id $CHAIN_ID

# Init two different chain nodes with two different folders
echo "Initializing chain nodes..."
echo "$KEY_NAME..."
./layerd init alicemoniker --chain-id $CHAIN_ID --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd init billmoniker --chain-id $CHAIN_ID --home ~/.layer/bill

# Add validator accounts
echo "Adding validator accounts..."
echo "$KEY_NAME..."
./layerd keys import-hex $KEY_NAME $PRIVATE_KEY_1 --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd keys import-hex bill $PRIVATE_KEY_2 --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill
echo "charlie..."
./layerd keys import-hex charlie $PRIVATE_KEY_3 --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME 
# import bill to alice's keyring for multisig
./layerd keys import-hex bill $PRIVATE_KEY_2 --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

# Create team multisig account
echo "Creating team multisig account..."
MULTISIG_NAME="team"
MULTISIG_THRESHOLD="2"
MULTISIG_MEMBERS="$KEY_NAME,bill"
./layerd keys add $MULTISIG_NAME --multisig="$MULTISIG_MEMBERS" --multisig-threshold=$MULTISIG_THRESHOLD --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "main..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "$KEY_NAME..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
echo "bill..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json


# Update signed_blocks_window in genesis.json for alice
echo "Updating signed_blocks_window in genesis.json for $KEY_NAME..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
echo "Updating signed_blocks_window in genesis.json for bill..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

# Update gov params in genesis.json
echo "Updating gov params in genesis.json..."
echo "main..."
jq '.app_state.gov.params.voting_period = "5m"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "1m"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "3m"' ~/.layer/config/genesis.json > temp.json && mv temp.json ~/.layer/config/genesis.json

echo "$KEY_NAME..."
jq '.app_state.gov.params.voting_period = "5m"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "1m"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "3m"' ~/.layer/$KEY_NAME/config/genesis.json > temp.json && mv temp.json ~/.layer/$KEY_NAME/config/genesis.json

echo "bill..."
jq '.app_state.gov.params.voting_period = "5m"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "1m"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "3m"' ~/.layer/bill/config/genesis.json > temp.json && mv temp.json ~/.layer/bill/config/genesis.json

# Create a tx to give alice loyas to stake
echo "Adding genesis accounts..."
echo "$KEY_NAME..."
./layerd genesis add-genesis-account $(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1)  10000000000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/bill

echo "team multisig..."
# ./layerd genesis add-genesis-account $(./layerd keys show $MULTISIG_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1) 5000000000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

echo "Add team address to dispute params..."
./layerd genesis add-team-account $(./layerd keys show $MULTISIG_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1) --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

#echo "charlie..."
#./layerd genesis add-genesis-account $(./layerd keys show charlie -a --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home ~/.layer/$KEY_NAME
# ./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend os --home ~/.layer/bill) 10000000000000loya --keyring-backend os --home ~/.layer/bill

# Create a tx to stake some loyas for alice
echo "Creating gentx $KEY_NAME..."
./layerd genesis gentx $KEY_NAME $STAKE_AMOUNT_1 --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1 --keyring-dir $LAYERD_NODE_HOME_1

# Add the transactions to the genesis block:q
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home $LAYERD_NODE_HOME_1

./layerd genesis validate-genesis --home $LAYERD_NODE_HOME_1

cp ~/.layer/alice/config/genesis.json ~/.layer/bill/config/genesis.json


# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for $KEY_NAME..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYERD_NODE_HOME_1/config/config.toml
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/bill/config/config.toml

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for $KEY_NAME..."
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" $LAYERD_NODE_HOME_1/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" ~/.layer/config/client.toml


echo "Start chain..."
# echo "password" |./layerd start --home $LAYERD_NODE_HOME_1 --api.enable --api.swagger --keyring-backend $KEYRING_BACKEND --key-name $KEY_NAME
./layerd start --home $LAYERD_NODE_HOME_1 --api.enable --api.swagger --keyring-backend $KEYRING_BACKEND --key-name $KEY_NAME

# ./layerd start --home ~/.layer/alice --api.enable --api.swagger --keyring-backend test --key-name alice

# Make alice a reporter
# ./layerd tx reporter create-reporter 0.1 1000000 alice --from alice --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice --keyring-dir ~/.layer/alice --fees 500loya --yes
# Bad Report
# ./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 --from alice --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice --keyring-dir ~/.layer/alice --fees 500loya --yes

# ----- Multisig 
# 1. Create a transaction (unsigned):
# ./layerd tx bank send team <recipient> 1000000loya --generate-only --from alice --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice > tx.json
#
# 2. Sign the transaction with alice:
# ./layerd tx sign tx.json --from alice --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice --multisig team --output-document tx-signed-alice.json
#
# 3. Sign the transaction with bill:
# ./layerd tx sign tx.json --from bill --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice --multisig team --output-document tx-signed-bill.json
#
# 4. Combine signatures and broadcast (both signatures required):
# ./layerd tx multisign tx.json team tx-signed-alice.json tx-signed-bill.json --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice > tx-final.json
# ./layerd tx broadcast tx-final.json --keyring-backend test --chain-id layertest-4 --home ~/.layer/alice
#


# ----- Gov Proposals
# 
# ./layerd tx gov submit-proposal mint_proposal.json \
#   --from alice --chain-id layertest-4 \
#   --keyring-backend test --home ~/.layer/alice \
#   --fees 500loya --yes

# {
#   "messages": [
#     {
#       "@type": "/layer.mint.MsgInit",
#       "authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx"
#     }
#   ],
#   "metadata": "mint initialization proposal",
#   "deposit": "1000000loya",
#   "title": "Initialize Mint Module",
#   "summary": "Initialize the mint module to start time-based rewards minting"
# }
# 
# ./layerd tx gov vote 1 yes --from alice --chain-id layertest-4 \
#   --keyring-backend test --home ~/.layer/alice --fees 500loya --yes