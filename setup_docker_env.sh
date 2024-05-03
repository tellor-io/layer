#!/bin/bash

# stop execution if any command fails
#set -e

KEYRING_BACKEND="test"
PASSWORD=password

echo "Clean up any existing docker images or containers..."
docker-compose -p layer-test down -v || true
docker image rm -f layerd_i || true
# docker image rm -f tmkms_alice || true
# docker image rm -f tmkms_bob || true

echo "Remove the old prod-sim files..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    rm -r -f ./prod-sim/$name
    mkdir -p ./prod-sim/$name
done


##  UNCOMMENT COMMANDS BELOW IF YOU NEED TO BUILD/REBUILD THE PROJECT

# rm -r -f build

# echo "Build with checksum using Makefile.."
# make build-with-checksum


# Build base image of layerd_i to be the image used across all containers
echo "Build base image of layerd_i to be the image used across all containers..."
docker build -f prod-sim/Dockerfile_layerd_alpine . -t layerd_i

# initialize the chain in all containers
echo "initialize the chain in all containers..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    layerd_i \
    init layer --chain-id layer
done

#   Create moniker for each node being ran in each container
echo "Create moniker for each node being ran in each container..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    layerd_i \
    init $name\moniker --chain-id layer --home /root/.layer/$name
done

# sets the denom to trb with a small unit of loya in the genesis file
echo "sets the denom to trb with a small unit of loya in the genesis file..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/"stake"/"loya"/g' /root/.layer/config/genesis.json

docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/"stake"/"loya"/g' /root/.layer/valAlice/config/genesis.json

# setup the config files to have a denom of trb and loya as the smallest unit
echo "setup the config files to have a denom of trb and loya as the smallest unit..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/([0-9]+)stake/\1loya/g' /root/.layer/$name/config/app.toml
done

for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/([0-9]+)stake/\1loya/g' /root/.layer/config/app.toml
done

#init the client.toml to have the chainId of layer
echo "init the client.toml to have the chainId of layer..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^chain-id = .*$/chain-id = "layer"/g' \
    /root/.layer/$name/config/client.toml
done

for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^chain-id = .*$/chain-id = "layer"/g' \
    /root/.layer/config/client.toml
done

#init the client.toml to have the KeyringBackend of variable
echo "init the client.toml to have the keyring-backend to env variable..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' \
    /root/.layer/$name/config/client.toml
done

for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' \
    /root/.layer/config/client.toml
done

# create keys for alice
echo "create validator key on alice desktop..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valAlice \
    add valAlice

# create keys for bob
echo "create validator key on bob desktop..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valBob   \
    add valBob

#Create keys for node carol 
echo "Create keys for nodeCarol..."
docker run --rm -it \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/nodeCarol \
    add nodeCarol

# set chain id in genesis file on alice desktop
echo "set chain id in genesis file on Alice desktop..."
docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' \
    /root/.layer/valAlice/config/genesis.json

docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' \
    /root/.layer/config/genesis.json

# Get address for nodeCarol
echo "Set address for nodeCarol to give them loya..."
NODE_CAROL=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/nodeCarol \
    show nodeCarol --address)

# give loya to nodeCarol
echo "give loya to nodeCarol..."
docker run --rm -it \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/nodeCarol \
    genesis add-genesis-account $NODE_CAROL 1000000000loya 

#move genesis file from carol to alice desktop
echo "move genesis file from carol to alice..."
mv prod-sim/nodeCarol/config/genesis.json \
    prod-sim/valAlice/config/

mv prod-sim/nodeCarol/nodeCarol/config/genesis.json \
    prod-sim/valAlice/valAlice/config/


#Get the address returned from the keyring on alice desktop
echo "Set address for alice to give them loya..."
ALICE=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valAlice \
    show valAlice --address)
echo $ALICE

# give loya to alice
echo "give loya to alice..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valAlice \
    genesis add-genesis-account $ALICE 10000000000000loya 

# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "Alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/valAlice/config/genesis.json > temp.json && mv temp.json prod-sim/valAlice/config/genesis.json
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/valAlice/valAlice/config/genesis.json > temp.json && mv temp.json prod-sim/valAlice/valAlice/config/genesis.json

echo "Bob...."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/valBob/config/genesis.json > temp.json && mv temp.json prod-sim/valBob/config/genesis.json
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/valBob/valBob/config/genesis.json > temp.json && mv temp.json prod-sim/valBob/valBob/config/genesis.json

#move genesis file from alice to bob desktop
echo "move genesis file from alice to bob..."
mv prod-sim/valAlice/config/genesis.json \
    prod-sim/valBob/config/

mv prod-sim/valAlice/valAlice/config/genesis.json \
    prod-sim/valBob/valBob/config/

# Gets Bobs address from his desktop to be used to send loya to him
echo "Gets Bobs address from his desktop to be used to send loya to him..."
BOB=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valBob \
    show valBob --address)
echo $BOB

#send loya to bobs account
echo "send loya to bobs account..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valBob \
    genesis add-genesis-account $BOB 10000000000000loya 

# Create gentx transaction for Bob to stake loya as validator
echo "Create gentx transaction for Bob to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    layerd_i \
    genesis gentx valBob 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valBob \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# move genesis file from bob to alice so that alice knows about bob
echo "move genesis file from bob to alice so that alice knows about bob"
mv prod-sim/valBob/config/genesis.json \
    prod-sim/valAlice/config/genesis.json
mv prod-sim/valBob/valBob/config/genesis.json \
    prod-sim/valAlice/valAlice/config/genesis.json

# create gentx tx for alice to stake loya as validator
echo "create gentx tx for alice to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    genesis gentx valAlice 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/valAlice \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# copy over gentx transaction so that alice has both the gentx transactions then verify
echo "copy over gentx transaction so that alice has both the gentx transactions then verify"
cp prod-sim/valBob/config/gentx/gentx-* \
    prod-sim/valAlice/config/gentx
cp prod-sim/valBob/valBob/config/gentx/gentx-* \
    prod-sim/valAlice/valAlice/config/gentx

echo "Collection gentxs in desk alice"
docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    genesis collect-gentxs --home /root/.layer/valAlice

# validate genesis file
echo "validate genesis file..."
docker run --rm -it \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    genesis validate-genesis --home /root/.layer/valAlice

# ensure all nodes have the same genesis file
echo "ensure all nodes have the same genesis file...."
for name in nodeCarol sentryAlice sentryBob valBob; do
    cp prod-sim/valAlice/config/genesis.json prod-sim/$name/config/genesis.json
    cp prod-sim/valAlice/valAlice/config/genesis.json prod-sim/$name/$name/config/genesis.json
done

# Get node info to be used in config values of other nodes
ALICE_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    layerd_i \
    --home /root/.layer/valAlice \
    comet show-node-id)

ALICE_IDENTIFIER=$ALICE_VAL_NODE_ID@valAlice:26656
echo $ALICE_IDENTIFIER

ALICE_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer \
    layerd_i \
    --home /root/.layer/sentryAlice \
    comet show-node-id)

ALICE_SENTRY_IDENTIFIER=$ALICE_SENTRY_NODE_ID@sentryAlice:26656
echo $ALICE_SENTRY_IDENTIFIER

BOB_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    layerd_i \
    --home /root/.layer/valBob \
    comet show-node-id)

BOB_IDENTIFIER=$BOB_VAL_NODE_ID@valBob:26656
echo $BOB_IDENTIFIER

BOB_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer \
    layerd_i \
    --home /root/.layer/sentryBob \
    comet show-node-id)

BOB_SENTRY_IDENTIFIER=$BOB_SENTRY_NODE_ID@sentryBob:26656
echo $BOB_SENTRY_IDENTIFIER


CAROL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    layerd_i \
    --home /root/.layer/nodeCarol \
    comet show-node-id)

CAROL_IDENTIFIER=$CAROL_NODE_ID@nodeCarol:26656
echo $CAROL_IDENTIFIER

ALICE_SENTRY_SEEDS=$BOB_SENTRY_IDENTIFIER,$CAROL_IDENTIFIER
BOB_SENTRY_SEEDS=$ALICE_SENTRY_IDENTIFIER,$CAROL_IDENTIFIER
CAROL_NODE_SEEDS=$ALICE_SENTRY_IDENTIFIER,$BOB_SENTRY_IDENTIFIER

#Update sentryAlice config.toml file
echo "Update sentryAlice config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_IDENTIFIER'"/g' /root/.layer/sentryAlice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$ALICE_SENTRY_SEEDS'"/g' /root/.layer/sentryAlice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$ALICE_VAL_NODE_ID'"/g' /root/.layer/sentryAlice/config/config.toml


# Update sentryBob config.toml file
echo "Update sentryBob config.toml file"
docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_IDENTIFIER'"/g' /root/.layer/sentryBob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$BOB_SENTRY_SEEDS'"/g' /root/.layer/sentryBob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$BOB_VAL_NODE_ID'"/g' /root/.layer/sentryBob/config/config.toml

# Update valAlice config.toml file
echo "Update valAlice config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_SENTRY_IDENTIFIER'"/g' /root/.layer/valAlice/config/config.toml

#Update valBob config.toml file
echo "Update valBob config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_SENTRY_IDENTIFIER'"/g' /root/.layer/valBob/config/config.toml

#Update nodeCarol config.toml file
echo "Update nodeCarol config.toml file.."
docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$CAROL_NODE_SEEDS'"/g' /root/.layer/nodeCarol/config/config.toml

# 127.0.0.1
# echo "Set api 

echo "Open up node carol to listen on all IPs"
docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' \
    /root/.layer/nodeCarol/config/config.toml

echo "enable api and swagger at localhost:1317"
docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enable = false/enable = true/g' \
    /root/.layer/nodeCarol/config/app.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^swagger = false/swagger = true/g' \
    /root/.layer/nodeCarol/config/app.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' \
    /root/.layer/nodeCarol/config/app.toml

#Update cors_allowed_origin
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' \
    /root/.layer/$name/config/config.toml
done

#Update enabled-unsafe-cors to true
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

# set timeout_commit or block time to 500ms
echo "Modifying timeout_commit in config.toml for alice..."
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/timeout_commit = "5s"/timeout_commit = "1s"/' /root/.layer/$name/config/config.toml
done

echo "Modifying timeout commit in root for all containers"
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer \
    --entrypoint sed \
    layerd_i \
    -Ei 's/timeout_commit = "5s"/timeout_commit = "1s"/' /root/.layer/config/config.toml
done

echo "Starting the chain in all containers..."
docker compose \
    --file ./prod-sim/docker-compose.yml \
    --project-name layer-test up \
    --detach

#mv ./build/layerd-linux-arm64 ./build/layerd

sleep 20

docker run --rm -it \
    --network layer-test_net-public \
    layerd_i status \
    --node "tcp://nodeCarol:26657"

docker run --rm -it \
    --network layer-test_net-public \
    layerd_i query staking validators --node "tcp://nodeCarol:26657"  > ./validator_info.yml 

ALICE_VAL_OP_ADD=$(yq '.validators[0].operator_address' ./validator_info.yml)
echo "ALICE: $ALICE_VAL_OP_ADD"
echo "Printing out val operator address for alice: $ALICE_VAL_OP_ADD"

echo "Gets Carols address from his desktop to be used to send loya to him"
CAROL=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/nodeCarol \
    show nodeCarol --address)
echo $CAROL

echo "Delegate from node carol to validator..."
docker run --rm -it \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
    --network layer-test_net-public \
    layerd_i tx staking delegate tellorvaloper1f8tupdprzqu38074xd4vasy0dqw98yeuykqayg 1000000loya \
    --keyring-backend test --home /root/.layer/nodeCarol \
    --chain-id layer --node "tcp://nodeCarol:26657" --from tellor10juyqjsv08vpsszm3uelp7ckfj3yfe34a4d2zw

echo "Now that you are delegated nodeCarol should start reporting now or soon..."



# ./layerd tx reporter create-reporter 1000000loya "{\"validatorAddress\": \"$ALICE_VAL_OP_ADD\", \"amount\": \"1000000loya\" }" \
#     --keyring-backend $KEYRING_BACKEND --home /root/.layer/nodeCarol \
#     --chain-id layer --node "tcp://nodeCarol:26657" --from $CAROL

# docker run --rm -it \
#     -v $(pwd)/prod-sim/nodeCarol:/root/.layer \
#     layerd_i keys list --home /root/.layer/nodeCarol --keyring-backend test