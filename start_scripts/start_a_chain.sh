#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

KEYRING_BACKEND="test"
PASSWORD="password"
KEY_NAME="alice"
CHAIN_ID="layer-local-1"
STAKE_AMOUNT_1="1000000000000loya"
PRIVATE_KEY_1="60d7de76caa85724ec588a5cd88a2afb77729658bcb783938109d2162779b225" # alice
PRIVATE_KEY_2="e40bf75172f36cb722a6db1042999f7e7b78b92b0d181cdd3a2dfb323595304a" # bill
PRIVATE_KEY_3="a6f6364de568b6a3ecfbf7d494852fc8114d52ef9a94faac987858efec3b1124" # charlie

export HOME_DIR="$HOME/.layer-chains/testnet/layer"
export LAYERD_NODE_HOME_1="$HOME_DIR/$KEY_NAME"

# Remove old test chains (if present)
echo "Removing old test chain data..."
rm -rf $HOME_DIR

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init $CHAIN_ID --chain-id $CHAIN_ID --home $HOME_DIR

# Init two different chain nodes with two different folders
echo "Initializing chain nodes..."
echo "$KEY_NAME..."
./layerd init alicemoniker --chain-id $CHAIN_ID --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd init billmoniker --chain-id $CHAIN_ID --home $HOME_DIR/bill

# Add validator accounts
echo "Adding validator accounts..."
echo "$KEY_NAME..."
./layerd keys import-hex $KEY_NAME $PRIVATE_KEY_1 --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd keys import-hex bill $PRIVATE_KEY_2 --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/bill
echo "charlie..."
./layerd keys import-hex charlie $PRIVATE_KEY_3 --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/$KEY_NAME 
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
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
echo "$KEY_NAME..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
echo "bill..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json


# Update signed_blocks_window in genesis.json for alice
echo "Updating signed_blocks_window in genesis.json for $KEY_NAME..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
jq '.app_state.slashing.params.signed_blocks_window = "1000"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
echo "Updating signed_blocks_window in genesis.json for bill..."
jq '.app_state.slashing.params.signed_blocks_window = "1000"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json

# Update gov params in genesis.json
echo "Updating gov params in genesis.json..."
echo "main..."
jq '.app_state.gov.params.voting_period = "1m"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "45s"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "30s"' $HOME_DIR/config/genesis.json > temp.json && mv temp.json $HOME_DIR/config/genesis.json

echo "$KEY_NAME..."
jq '.app_state.gov.params.voting_period = "1m"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "45s"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "30s"' $HOME_DIR/$KEY_NAME/config/genesis.json > temp.json && mv temp.json $HOME_DIR/$KEY_NAME/config/genesis.json

echo "bill..."
jq '.app_state.gov.params.voting_period = "1m"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json
jq '.app_state.gov.params.max_deposit_period = "45s"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].denom = "loya"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json
jq '.app_state.gov.params.min_deposit[0].amount = "100"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json
jq '.app_state.gov.params.expedited_voting_period = "30s"' $HOME_DIR/bill/config/genesis.json > temp.json && mv temp.json $HOME_DIR/bill/config/genesis.json

# Create a tx to give alice loyas to stake
echo "Adding genesis accounts..."
echo "$KEY_NAME..."
./layerd genesis add-genesis-account $(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1)  10000000000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/bill) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/bill

echo "team multisig..."
# ./layerd genesis add-genesis-account $(./layerd keys show $MULTISIG_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1) 5000000000000loya --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

echo "Add team address to dispute params..."
./layerd genesis add-team-account $(./layerd keys show $MULTISIG_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1) --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1

#echo "charlie..."
#./layerd genesis add-genesis-account $(./layerd keys show charlie -a --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/$KEY_NAME) 10000000000000loya --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/$KEY_NAME
# ./layerd genesis add-genesis-account $(./layerd keys show bill -a --keyring-backend os --home $HOME_DIR/bill) 10000000000000loya --keyring-backend os --home $HOME_DIR/bill

# Create a tx to stake some loyas for alice
echo "Creating gentx $KEY_NAME..."
./layerd genesis gentx $KEY_NAME $STAKE_AMOUNT_1 --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1 --keyring-dir $LAYERD_NODE_HOME_1

# Add the transactions to the genesis block:q
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home $LAYERD_NODE_HOME_1

./layerd genesis validate-genesis --home $LAYERD_NODE_HOME_1

cp $HOME_DIR/alice/config/genesis.json $HOME_DIR/bill/config/genesis.json

# 1.5 second block times
# Modify timeout_commit in config.toml for alice
echo "Modifying timeout_commit in config.toml for $KEY_NAME..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "2s"/' $LAYERD_NODE_HOME_1/config/config.toml
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "2s"/' $HOME_DIR/bill/config/config.toml

# Modify keyring-backend in client.toml for alice
echo "Modifying keyring-backend in client.toml for $KEY_NAME..."
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" $LAYERD_NODE_HOME_1/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' "s/keyring-backend = \"test\"/keyring-backend = \"$KEYRING_BACKEND\"/" $HOME_DIR/config/client.toml


echo "Start chain..."
# echo "password" |./layerd start --home $LAYERD_NODE_HOME_1 --api.enable --api.swagger --keyring-backend $KEYRING_BACKEND --key-name $KEY_NAME
./layerd start --home $LAYERD_NODE_HOME_1 --api.enable --api.swagger --keyring-backend $KEYRING_BACKEND --key-name $KEY_NAME

# ./layerd start --home $HOME_DIR/alice --api.enable --api.swagger --keyring-backend test --key-name alice

# ----- Reporting
# Make alice a reporter
# ./layerd tx reporter create-reporter 0.1 1000000 alice --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --keyring-dir $HOME_DIR/alice --fees 500loya --yes
#
# Bad cycle list report
# ./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --keyring-dir $HOME_DIR/alice --fees 500loya --yes
#
# build reporter daemon
# 
# 
# run reporter daemon
# ./reporterd --chain-id layer-local-1 --grpc-addr 0.0.0.0:9090 --from alice --home $HOME_DIR/alice --keyring-backend test --node tcp://0.0.0.0:26657
#
# tip random query
# ./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004706f6f700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
#   10000loya \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes
# # 
# # submit for random query
# ./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004706f6f700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
#   000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes
# #
# # tip submit LTC / USD
# ./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
#   10000loya \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes

# ./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
#   000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes
# 
# tip sFRXUSD / USD
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000007736672787573640000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes
# 
# ----- Multisig 
# 1. Create a transaction (unsigned):
# ./layerd tx bank send team <recipient> 1000000loya --generate-only --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice > tx.json
#
# 2. Sign the transaction with alice:
# ./layerd tx sign tx.json --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --multisig team --output-document tx-signed-alice.json
#
# 3. Sign the transaction with bill:
# ./layerd tx sign tx.json --from bill --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --multisig team --output-document tx-signed-bill.json
#
# 4. Combine signatures and broadcast (both signatures required):
# ./layerd tx multisign tx.json team tx-signed-alice.json tx-signed-bill.json --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice > tx-final.json
# ./layerd tx broadcast tx-final.json --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice


# ----- Gov Proposals
# ./layerd tx gov submit-proposal mint_proposal.json \
#   --from alice --chain-id layer-local-1 \
#   --keyring-backend test --home $HOME_DIR/alice \
#   --fees 500loya --yes
#
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
#   "summary": "Initialize the mint module to start time-based rewards minting!!"
# }
# 
# ./layerd tx gov vote 1 yes --from alice --chain-id layer-local-1 \
#   --keyring-backend test --home $HOME_DIR/alice --fees 500loya --yes
#
# Change extra rewards rate
#
# Send module account 100 trb
# ./layerd tx bank send alice tellor17wa2hwtrmfdytr4fnfueajalhr468tmu0p2h58 100000000loya --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 1500loya \
#   --gas 500000 \
#   --yes
#
#
# submit proposal with update_extra_rewards_rate.json
# {
#   "messages": [
#     {
#       "@type": "/layer.mint.MsgUpdateExtraRewardRate",
#       "authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx",
#       "daily_extra_rewards": "200000000",
#       "bond_denom": "loya"
#     }
#   ],
#   "metadata": "ipfs://YourIPFSHashHere",
#   "deposit": "1000000loya",
#   "title": "Update Extra Rewards Rate",
#   "summary": "Increase the daily extra rewards rate from the current default (146,940,000 loya/day) to 200,000,000 loya/day to provide additional incentives for oracle reporters"
# }
 
# ./layerd tx gov submit-proposal update_extra_reward_rate_proposal.json \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --fees 500loya \
#   --yes
#
# ./layerd tx gov vote 2 yes \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --fees 500loya \
#   --yes


# ----- EVMCall
# {
#     "document_hash": "EVMCall",
#     "query_type": "EVMCall",
#     "response_value_type": "bytes",
#     "abi_components": [
#         {
#             "name": "chainId",
#             "field_type": "uint256"
#         },
#         {
#             "name": "contractAddress",
#             "field_type": "address"
#         },
#         {
#             "name": "calldata",
#             "field_type": "bytes"
#         }
#     ],
#     "aggregation_method": "weighted-mode",
#     "registrar": "tellor1alcefjzkk37qmfrnel8q4eruyll0pc8arxhxxw",
#     "report_block_window": 10
# }
# ./layerd tx registry register-spec EVMCall evmcall_dataspec.json \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes

# Generate evm call query data for mainnet tokenbridge depositId read
# ./layerd query registry generate-querydata EVMCall '["1", "0x5589e306b1920f009979a50b88cae32aecd471e4", "0x9852099c"]'
#  00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000049852099c00000000000000000000000000000000000000000000000000000000

# Tip 
#  ./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000049852099c00000000000000000000000000000000000000000000000000000000 \
#   1000000loya \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes

# Submit
# ./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000049852099c00000000000000000000000000000000000000000000000000000000 \
#   00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000068e71aa20000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000004d \
#   --from alice \
#   --chain-id layer-local-1 \
#   --keyring-backend test \
#   --home $HOME_DIR/alice \
#   --keyring-dir $HOME_DIR/alice \
#   --fees 500loya \
#   --yes