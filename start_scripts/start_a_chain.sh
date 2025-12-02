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
echo "Creating Alice and Bill keys..."
echo "$KEY_NAME..."
./layerd keys import-hex $KEY_NAME $PRIVATE_KEY_1 --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1
echo "bill..."
./layerd keys import-hex bill $PRIVATE_KEY_2 --keyring-backend $KEYRING_BACKEND --home $HOME_DIR/bill

# import bill to alice's keyring for multisig
echo "importing bill to alice's keyring for multisig..."
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

# Create a tx to stake some loyas for alice
echo "Creating gentx $KEY_NAME..."
./layerd genesis gentx $KEY_NAME $STAKE_AMOUNT_1 --chain-id $CHAIN_ID --keyring-backend $KEYRING_BACKEND --home $LAYERD_NODE_HOME_1 --keyring-dir $LAYERD_NODE_HOME_1

# Add the transactions to the genesis block:q
echo "Collecting gentxs..."
./layerd genesis collect-gentxs --home $LAYERD_NODE_HOME_1

./layerd genesis validate-genesis --home $LAYERD_NODE_HOME_1

cp $HOME_DIR/alice/config/genesis.json $HOME_DIR/bill/config/genesis.json

# 2 second block times
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

# --------------------------------------------------------------------------------------------------------------
# END OF SCRIPT ------------------------------------------------------------------------------------------------
# --------------------------------------------------------------------------------------------------------------





# --------------------------------------------------------------------------------------------------------------
# --------------------------------------------------------------------------------------------------------------
# QUICK COMMANDS -----------------------------------------------------------------------------------------------
# --------------------------------------------------------------------------------------------------------------
# --------------------------------------------------------------------------------------------------------------

export HOME_DIR="$HOME/.layer-chains/testnet/layer"

#  start the chain again
./layerd start --home $HOME_DIR/alice --api.enable --api.swagger --keyring-backend test --key-name alice

# Reporting ----------------------------------------------------------------------------------------------------
# alice create reporter
./layerd tx reporter create-reporter 0.1 1000000 alice --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --keyring-dir $HOME_DIR/alice --fees 500loya --yes

# bad eth price
./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --keyring-dir $HOME_DIR/alice --fees 500loya --yes

# build reporter daemon
cd daemons && make build

# run reporter daemon
./reporterd \
  --chain-id layer-local-1 \
  --grpc-addr 0.0.0.0:9090 \
  --from alice \
  --home $HOME_DIR/alice \
  --keyring-backend test \
  --node tcp://0.0.0.0:26657 \

# add to ^ for price guard (adjust numbers as needed) 
  --price-guard-enabled=true \
  --price-guard-threshold=0.5 \
  --price-guard-max-age=30m \

# manual bad ltc price
./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

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

# query sfrxusd reports
./layerd query oracle get-reportsby-qid ab30caa3e7827a27c153063bce02c0b260b29c0c164040c003f0f9ec66002510

#  tip xyz/usd (junk)
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000378797a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# submit xyz/usd
./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000378797a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  000000000000000000000000000000000000000000000084bd26b6c2dd7c0000 \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# query xyz reports
./layerd query oracle get-reportsby-qid 917ea118ee92837ddd2c3dcd33fb21064285f9bfab13bc2ce75372ab40608623

# tip susn/usd
  ./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000047375736e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# tip reth/usd
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004726574680000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# tip wsteth/usd
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000006777374657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# tip king/usd
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000046b696e670000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# tip saga/usd
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004736167610000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
   --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# tip sfrxusd/usd
./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000007736672787573640000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 \
  10000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# evmcall_ds.json
{
    "document_hash": "EVMCall",
    "query_type": "EVMCall",
    "response_value_type": "bytes",
    "abi_components": [
        {
            "name": "chainId",
            "field_type": "uint256"
        },
        {
            "name": "contractAddress",
            "field_type": "address"
        },
        {
            "name": "calldata",
            "field_type": "bytes"
        }
    ],
    "aggregation_method": "weighted-mode",
    "registrar": "tellor1alcefjzkk37qmfrnel8q4eruyll0pc8arxhxxw",
    "report_block_window": 10
}

# register evmcall
./layerd tx registry register-spec EVMCall start_scripts/test_jsons/evmcall_ds.json \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# nflwinner_ds.json
{
    "document_hash": "NFLWinner",
    "query_type": "NFLWinner",
    "response_value_type": "bytes",
    "abi_components": [
        {
            "name": "week",
            "field_type": "uint256"
        },
        {
            "name": "team1",
            "field_type": "string"
        },
        {
            "name": "team2",
            "field_type": "string"
        }
    ],
    "aggregation_method": "weighted-mode",
    "registrar": "tellor1alcefjzkk37qmfrnel8q4eruyll0pc8arxhxxw",
    "report_block_window": 10
}

# register nflwinner
./layerd tx registry register-spec NFLWinner start_scripts/test_jsons/nflwinner_ds.json \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# Generate evm call query data for mainnet tokenbridge PAUSE_PERIOD (0x7f629c65)
./layerd query registry generate-querydata EVMCall '["1", "0x5589e306b1920f009979a50b88cae32aecd471e4", "0x7f629c65"]'
 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000047f629c6500000000000000000000000000000000000000000000000000000000

# tip evmcall ["1", "0x5589e306b1920f009979a50b88cae32aecd471e4", "0x7f629c65"]
 ./layerd tx oracle tip 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000047f629c6500000000000000000000000000000000000000000000000000000000 \
  1000000loya \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# submit evmcall ["1", "0x5589e306b1920f009979a50b88cae32aecd471e4", "0x7f629c65"]
./layerd tx oracle submit-value 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000047f629c6500000000000000000000000000000000000000000000000000000000 \
  00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000068e71aa2000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000001bad00 \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --keyring-dir $HOME_DIR/alice \
  --fees 500loya \
  --yes

# Gov Proposals ------------------------------------------------------------------------------------------------
# mint_proposal.json
{
  "messages": [
    {
      "@type": "/layer.mint.MsgInit",
      "authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx"
    }
  ],
  "metadata": "mint initialization proposal -- test",
  "deposit": "1000000loya",
  "title": "Initialize Mint Module",
  "summary": "Initialize the mint module to start time-based rewards minting!!"
}

# turn on minting
./layerd tx gov submit-proposal start_scripts/test_jsons/mint_init.json \
  --from alice --chain-id layer-local-1 \
  --keyring-backend test --home $HOME_DIR/alice --fees 500loya --yes

# vote on minting proposal
./layerd tx gov vote 1 yes --from alice --chain-id layer-local-1 \
  --keyring-backend test --home $HOME_DIR/alice --fees 500loya --yes


# update_extra_rewards_rate.json
{
  "messages": [
    {
      "@type": "/layer.mint.MsgUpdateExtraRewardRate",
      "authority": "tellor10d07y265gmmuvt4z0w9aw880jnsr700j6527vx",
      "daily_extra_rewards": "36735000",
      "bond_denom": "loya"
    }
  ],
  "metadata": "test",
  "deposit": "1000000loya",
  "title": "Update Extra Rewards Rate",
  "summary": "Lower the daily extra rewards rate from the current default (146,940,000 loya/day) to 36,735,000 loya/day"
}
 
./layerd tx gov submit-proposal start_scripts/test_jsons/update_extra_rewards.json \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --fees 500loya \
  --yes
#
./layerd tx gov vote 2 yes \
  --from alice \
  --chain-id layer-local-1 \
  --keyring-backend test \
  --home $HOME_DIR/alice \
  --fees 500loya \
  --yes


# Multisig ----------------------------------------------------------------------------------------------------
# 1. Create a transaction (unsigned):
./layerd tx bank send team <recipient> 1000000loya --generate-only --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice > tx.json

# 2. Sign the transaction with alice:
./layerd tx sign tx.json --from alice --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --multisig team --output-document tx-signed-alice.json

# 3. Sign the transaction with bill:
./layerd tx sign tx.json --from bill --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice --multisig team --output-document tx-signed-bill.json

# 4. Combine signatures and broadcast (both signatures required):
./layerd tx multisign tx.json team tx-signed-alice.json tx-signed-bill.json --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice > tx-final.json
./layerd tx broadcast tx-final.json --keyring-backend test --chain-id layer-local-1 --home $HOME_DIR/alice