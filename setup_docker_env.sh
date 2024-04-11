#!/bin/bash

# stop execution if any command fails
#set -e

KEYRING_BACKEND="os"
PASSWORD="password"

echo "Clean up any existing docker images or containers"
docker-compose -p layer-test down -v || true
docker image rm -f layerd_i || true

echo "Remove the old prod-sim files"
for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    rm -r -f ./prod-sim/$name
    mkdir -p ./prod-sim/$name
done

#rm -r -f build

# echo "Build with checksum using Makefile.."
# make build-with-checksum

# Build base image of layerd_i to be the image used across all containers
echo "Build base image of layerd_i to be the image used across all containers"
docker build -f prod-sim/Dockerfile-layerd-alpine . -t layerd_i

# initialize the chain in all containers
echo "initialize the chain in all containers"
for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    layerd_i \
    init layer --chain-id layer --home /root/.layer/$name
done


# sets the denom to trb with a small unit of loya in the genesis file
echo "sets the denom to trb with a small unit of loya in the genesis file"
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer/desk-alice \
    --entrypoint sed \
    layerd_i \
    -i 's/"trb"/"loya"/g' /root/.layer/desk-alice/config/genesis.json

# setup the config files to have a denom of trb and loya as the smallest unit
echo "setup the config files to have a denom of trb and loya as the smallest unit"
for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/([0-9]+)trb/\1loya/g' /root/.layer/$name/config/app.toml
done

#init the client.toml to have the chainId of layer
echo "init the client.toml to have the chainId of layer"
for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^chain-id = .*$/chain-id = "layer"/g' \
    /root/.layer/$name/config/client.toml
done

#init the client.toml to have the KeyringBackend of variable
echo "init the client.toml to have the keyring-backend to env variable"
# for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
#     docker run --rm -i \
#     -v $(pwd)/prod-sim/$name:/root/.layer/$name \
#     --entrypoint sed \
#     layerd_i \
#     -Ei 's/^keyring-backend = .*"/keyring-backend = "'$KEYRING_BACKEND'"/g' \
#     /root/.layer/config/client.toml
# done

# create validator key on alice desktop
echo "create validator key on alice desktop"
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer/desk-alice \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --keyring-dir /root/.layer/desk-alice --home /root/.layer/desk-alice \
    add desk-alice

# move password used in key creation to file (DO NOT DO THIS IN PROD)
echo "move password used in key creation to file DO NOT DO THIS IN PROD"
echo -n password > prod-sim/desk-alice/passphrase.txt

# create validator key on bob desktop
echo "create validator key on bob desktop"
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-bob:/root/.layer/desk-bob \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --keyring-dir /root/.layer/desk-bob --home /root/.layer/desk-bob   \
    add desk-bob

# move password used in key creation to file (DO NOT DO THIS IN PROD)
echo "move password used in key creation to file DO NOT DO THIS IN PROD"
echo -n $PASSWORD > prod-sim/desk-bob/passphrase.txt

# set chain id in genesis file on Alice desktop
echo "set chain id in genesis file on Alice desktop"
docker run --rm -i \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer/desk-alice \
    --entrypoint sed \
    layerd_i \
    -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' \
    /root/.layer/desk-alice/config/genesis.json


#Set the address returned from the keyring on alice desktop
echo "Set the address returned from the keyring on alice desktop"
ALICE=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer/desk-alice \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --keyring-dir /root/.layer/desk-alice --home /root/.layer/desk-alice \
    show desk-alice --address)

# give loya to alice
echo "give loya to alice..."
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer/desk-alice \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/desk-alice \
    genesis add-genesis-account $ALICE 10000000000000loya 

# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "Alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/desk-alice/config/genesis.json > temp.json && mv temp.json prod-sim/desk-alice/config/genesis.json

#move genesis file from alice to bob desktop
echo "move genesis file from alice to bob desktop"
mv prod-sim/desk-alice/config/genesis.json \
    prod-sim/desk-bob/config/

# Gets Bobs address from his desktop to be used to send loya to him
echo "Gets Bobs address from his desktop to be used to send loya to him"
BOB=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/desk-bob:/root/.layer/desk-bob \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/desk-bob \
    show desk-bob --address)
echo $BOB

#send loya to bobs account
echo "send loya to bobs account"
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-bob:/root/.layer/desk-bob \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/desk-bob \
    genesis add-genesis-account $BOB 10000000000000loya 

# Copy validator keys from val nodes to bob and alice desktop
echo "Copy validator keys from val nodes to bob and alice desktop"
cp prod-sim/val-alice/config/priv_validator_key.json \
    prod-sim/desk-alice/config/priv_validator_key.json

cp prod-sim/val-bob/config/priv_validator_key.json \
    prod-sim/desk-bob/config/priv_validator_key.json

# Create gentx transaction for Bob to stake loya as validator
echo "Create gentx transaction for Bob to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/desk-bob:/root/.layer/desk-bob \
    layerd_i \
    genesis gentx desk-bob 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --keyring-dir /root/.layer/desk-bob --home /root/.layer/desk-bob \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# move genesis file from bob to alice so that alice knows about bob
echo "move genesis file from bob to alice so that alice knows about bob"
mv prod-sim/desk-bob/config/genesis.json \
    prod-sim/desk-alice/config/genesis.json

# create gentx tx for alice to stake loya as validator
echo "create gentx tx for alice to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer \
    layerd_i \
    genesis gentx desk-alice 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --keyring-dir /root/.layer --home /root/.layer \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# copy over gentx transaction so that alice has both the gentx transactions then verify
echo "copy over gentx transaction so that alice has both the gentx transactions then verify"
cp prod-sim/desk-bob/config/gentx/gentx-* \
    prod-sim/desk-alice/config/gentx

echo "Collection gentxs in desk alice"
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer \
    layerd_i \
    genesis collect-gentxs

# validate genesis file
echo "validate genesis file..."
docker run --rm -it \
    -v $(pwd)/prod-sim/desk-alice:/root/.layer \
    layerd_i \
    genesis validate-genesis

# ensure all nodes have the same genesis file
echo "ensure all nodes have the same genesis file...."
for name in desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    cp prod-sim/desk-alice/config/genesis.json prod-sim/$name/config/genesis.json
done

# Get node info to be used in config values of other nodes
ALICE_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/val-alice:/root/.layer \
    layerd_i \
    --home /root/.layer \
    comet show-node-id)

ALICE_IDENTIFIER=$ALICE_VAL_NODE_ID@val-alice:26656
echo $ALICE_IDENTIFIER

ALICE_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-alice:/root/.layer \
    layerd_i \
    --home /root/.layer \
    comet show-node-id)

ALICE_SENTRY_IDENTIFIER=$ALICE_SENTRY_NODE_ID@sentry-alice:26656
echo $ALICE_SENTRY_IDENTIFIER

BOB_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/val-bob:/root/.layer \
    layerd_i \
    --home /root/.layer \
    comet show-node-id)

BOB_IDENTIFIER=$BOB_VAL_NODE_ID@val-bob:26656
echo $BOB_IDENTIFIER

BOB_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-bob:/root/.layer \
    layerd_i \
    --home /root/.layer \
    comet show-node-id)

BOB_SENTRY_IDENTIFIER=$BOB_SENTRY_NODE_ID@sentry-bob:26656
echo $BOB_SENTRY_IDENTIFIER


CAROL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/node-carol:/root/.layer \
    layerd_i \
    --home /root/.layer \
    comet show-node-id)

CAROL_IDENTIFIER=$CAROL_NODE_ID@node-carol:26656
echo $CAROL_IDENTIFIER

ALICE_SENTRY_SEEDS=$BOB_SENTRY_IDENTIFIER,$CAROL_IDENTIFIER
BOB_SENTRY_SEEDS=$ALICE_SENTRY_IDENTIFIER,$CAROL_IDENTIFIER
CAROL_NODE_SEEDS=$ALICE_SENTRY_IDENTIFIER,$BOB_SENTRY_IDENTIFIER

#Update sentry-alice config.toml file
echo "Update sentry-alice config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-alice:/root/.layer/sentry-alice \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_IDENTIFIER'"/g' /root/.layer/sentry-alice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-alice:/root/.layer/sentry-alice \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$ALICE_SENTRY_SEEDS'"/g' /root/.layer/sentry-alice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-alice:/root/.layer/sentry-alice \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$ALICE_VAL_NODE_ID'"/g' /root/.layer/sentry-alice/config/config.toml


# Update sentry-bob config.toml file
echo "Update sentry-bob config.toml file"
docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-bob:/root/.layer/sentry-bob \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_IDENTIFIER'"/g' /root/.layer/sentry-bob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-bob:/root/.layer/sentry-bob \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$BOB_SENTRY_SEEDS'"/g' /root/.layer/sentry-bob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentry-bob:/root/.layer/sentry-bob \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$BOB_VAL_NODE_ID'"/g' /root/.layer/sentry-bob/config/config.toml

# Update val-alice config.toml file
echo "Update val-alice config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/val-alice:/root/.layer/val-alice \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_SENTRY_IDENTIFIER'"/g' /root/.layer/val-alice/config/config.toml

#Update val-bob config.toml file
echo "Update val-bob config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/val-bob:/root/.layer/val-bob \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_SENTRY_IDENTIFIER'"/g' /root/.layer/val-bob/config/config.toml

#Update node-carol config.toml file
echo "Update node-carol config.toml file.."
docker run --rm -i \
    -v $(pwd)/prod-sim/node-carol:/root/.layer/node-carol \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$CAROL_NODE_SEEDS'"/g' /root/.layer/node-carol/config/config.toml

#Update cors_allowed_origin
for name in node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' \
    /root/.layer/$name/config/config.toml
done

#Update enabled-unsafe-cors to true
for name in node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

for name in node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

# set timeout_commit or block time to 500ms
echo "Modifying timeout_commit in config.toml for alice..."
for name in desk-alice desk-bob node-carol sentry-alice sentry-bob val-alice val-bob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/timeout_commit = "5s"/timeout_commit = "500ms"/' /root/.layer/$name/config/config.toml
done

echo "Starting the chain in all containers..."
docker compose \
    --file ./prod-sim/docker-compose.yml \
    --project-name layer-test up


# Export alice key from os backend and import to test backend
# echo "Exporting alice key..."
# echo $PASSWORD | docker run --rm -i \
#     -v $(pwd)/prod-sim/desk-alice:/root/.layer \
#     layerd_i \
#     keys \
#     --keyring-backend $KEYRING_BACKEND --home /root/.layer \
#     export desk-alice > /root/.layer/alice_keyfile

# echo "Importing alice key to test backend..."
# echo $PASSWORD | docker run --rm -i \
#     -v $(pwd)/prod-sim/desk-alice:/root/.layer \
#     layerd_i \
#     keys \
#     --keyring-backend test --home /root/.layer \
#     import desk-alice /root/.layer/alice_keyfile


# # Export bob key from os backend and import to test backend
# echo "Exporting bob key..."
# echo $PASSWORD | docker run --rm -i \
#     -v $(pwd)/prod-sim/desk-bob:/root/.layer \
#     layerd_i \
#     keys \
#     --keyring-backend $KEYRING_BACKEND --home /root/.layer \
#     export desk-bob > /root/.layer/desk-bob/bob_keyfile

# echo "Importing bob key to test backend..."
# echo $PASSWORD | docker run --rm -i \
#     -v $(pwd)/prod-sim/desk-bob:/root/.layer \
#     layerd_i \
#     keys \
#     --keyring-backend test --home /root/.layer \
#     import desk-bob /root/.layer/desk-bob/bob_keyfile
