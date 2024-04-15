#!/bin/bash

# stop execution if any command fails
#set -e

KEYRING_BACKEND="test"
PASSWORD=password

echo "Clean up any existing docker images or containers"
docker-compose -p layer-test down -v || true
docker image rm -f layerd_i || true
docker image rm -f tmkms_alice || true
docker image rm -f tmkms_bob || true

echo "Remove the old prod-sim files"
for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob kmsBob kmsAlice; do
    rm -r -f ./prod-sim/$name
    mkdir -p ./prod-sim/$name
done



rm -r -f build

echo "Build with checksum using Makefile.."
make build-with-checksum

#mv ./prod-sim/Dockerfile_tmkms ./prod-sim/Dockerfil_tmkms.txt

# Build base image of layerd_i to be the image used across all containers
echo "Build base image of layerd_i to be the image used across all containers"
docker build -f prod-sim/Dockerfile_layerd_alpine . -t layerd_i

# mv ./prod-sim/Dockerfil_tmkms.txt ./prod-sim/Dockerfile_tmkms 
# mv ./prod-sim/Dockerfile_layerd_alpine ./prod-sim/Dockerfil_layerd_alpine.txt

# echo "Build the KMS used for validator key management and signing ONLY NEED TO RUN THIS ONCE"
# docker build -f prod-sim/Dockerfile_tmkms . -t tmkms_alice

# echo "Build the KMS used for validator key management and signing"
# docker build -f prod-sim/Dockerfile_tmkms . -t tmkms_bob

# mv ./prod-sim/Dockerfil_layerd_alpine.txt ./prod-sim/Dockerfile_layerd_alpine

# initialize the chain in all containers
echo "initialize the chain in all containers"
for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    layerd_i \
    init layer --chain-id layer --home /root/.layer/$name
done

# sets the denom to trb with a small unit of loya in the genesis file
echo "sets the denom to trb with a small unit of loya in the genesis file"
docker run --rm -it \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    --entrypoint sed \
    layerd_i \
    -i 's/"stake"/"loya"/g' /root/.layer/deskAlice/config/genesis.json

# setup the config files to have a denom of trb and loya as the smallest unit
echo "setup the config files to have a denom of trb and loya as the smallest unit"
for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/([0-9]+)stake/\1loya/g' /root/.layer/$name/config/app.toml
done

#init the client.toml to have the chainId of layer
echo "init the client.toml to have the chainId of layer"
for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^chain-id = .*$/chain-id = "layer"/g' \
    /root/.layer/$name/config/client.toml
done

#init the client.toml to have the KeyringBackend of variable
echo "init the client.toml to have the keyring-backend to env variable"
# for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
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
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskAlice \
    add valAlice

# move password used in key creation to file (DO NOT DO THIS IN PROD)
echo "move password used in key creation to file DO NOT DO THIS IN PROD"
echo -n password > prod-sim/deskAlice/passphrase.txt

# echo "Initiliaze a kms image for alice"
# docker run --rm -it \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#     tmkms_alice \
#     init /root/tmkms_alice

# echo "Set proper version of CometBFT package for tmkms"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms \
#     --entrypoint sed \
#     tmkms_alice \
#     -i 's/protocol_version = "v0.34"/protocol_version = "v0.38"/g' /root/tmkms/tmkms.toml

# echo "Set Name of file where alice key will be"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#   --entrypoint sed \
#   tmkms_alice \
#   -Ei 's/path = "\/root\/tmkms_alice\/secrets\/cosmoshub-3-consensus.key"/path = "\/root\/tmkms_alice\/secrets\/valAlice_consensus.key"/g' \
#   /root/tmkms_alice/tmkms.toml

# echo "Update chain id in tmkms.toml to layer"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#   --entrypoint sed \
#   tmkms_alice \
#   -i 's/id = "cosmoshub-3"/id = "layer"/g' /root/tmkms_alice/tmkms.toml

# echo "Update the path to the state file to represent the correct chain id"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#   --entrypoint sed \
#   tmkms_alice \
#   -i 's/state_file = "\/root\/tmkms_alice\/state\/cosmoshub-3-consensus.json"/state_file = "\/root\/tmkms_alice\/state\/priv_validator_state.json"/g' /root/tmkms_alice/tmkms.toml

# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#   --entrypoint sed \
#   tmkms_alice \
#   -i 's/chain_ids = \["cosmoshub-3"\]/chain_ids = \["layer"\]/g' /root/tmkms_alice/tmkms.toml

# echo "Copy public consensus key to valAlice"
# docker run --rm -t \
#     -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
#     layerd_i \
#     comet show-validator \
#     | tr -d '\n' | tr -d '\r' \
#     > prod-sim/deskAlice/config/pub_validator_key_valAlice.json


# echo "Moving priv validator key off of valAlice"
# cp prod-sim/valAlice/config/priv_validator_key.json \
#   prod-sim/deskAlice/config/priv_validator_key_valAlice.json

# mv prod-sim/valAlice/config/priv_validator_key.json \
#   prod-sim/kmsAlice/secrets/priv_validator_key_valAlice.json

# echo "import validator key into tmkms softsign feature"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#     -w /root/tmkms_alice \
#     tmkms_alice \
#     softsign import secrets/priv_validator_key_valAlice.json \
#     secrets/valAlice_consensus.key

# echo "copy over validator key from sentryAlice where it is empty to remove alice's info"
# cp prod-sim/sentryAlice/config/priv_validator_key.json \
#     prod-sim/valAlice/config/


# echo "Set Port that will be used to communicate with val alice over private connection"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#     --entrypoint sed \
#     tmkms_alice \
#     -Ei 's/^addr = "tcp:.*$/addr = "tcp:\/\/valAlice:26699"/g' /root/tmkms_alice/tmkms.toml

# echo "Update the key format in tmkms"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#     --entrypoint sed \
#     tmkms_alice \
#     -Ei 's/account_key_prefix = "cosmospub"/account_key_prefix = "tellor"/g' /root/tmkms_alice/tmkms.toml

# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsAlice:/root/tmkms_alice \
#     --entrypoint sed \
#     tmkms_alice \
#     -Ei 's/consensus_key_prefix = "cosmosvalconspub"/consensus_key_prefix = "tellorvalconspub"/g' /root/tmkms_alice/tmkms.toml

# echo "Inform valAlice of the port to listen for the tmkms"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/priv_validator_laddr = ""/priv_validator_laddr = "tcp:\/\/0.0.0.0:26699"/g' \
#   /root/.layer/valAlice/config/config.toml

# echo "valAlice config to not look for consensus key"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/^priv_validator_key_file/# priv_validator_key_file/g' \
#   /root/.layer/valAlice/config/config.toml

# echo "Comment out validator state file so valAlice no longer looks for it"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/^priv_validator_state_file/# priv_validator_state_file/g' \
#   /root/.layer/valAlice/config/config.toml

# cp prod-sim/sentryAlice/config/priv_validator_key.json \
#     prod-sim/valAlice/config/



# create validator key on bob desktop
echo "create validator key on bob desktop"
docker run --rm -it \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskBob   \
    add valBob

# move password used in key creation to file (DO NOT DO THIS IN PROD)
echo "move password used in key creation to file DO NOT DO THIS IN PROD"
echo -n $PASSWORD > prod-sim/deskBob/passphrase.txt

# echo "Initiliaze a kms image for bob"
# docker run --rm -it \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#     tmkms_bob \
#     init /root/tmkms_bob

# echo "Set proper version of CometBFT package for tmkms"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms \
#     --entrypoint sed \
#     tmkms_alice \
#     -i 's/protocol_version = "v0.34"/protocol_version = "v0.38"/g' /root/tmkms/tmkms.toml

# echo "Set Name of file where bob key will be"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#   --entrypoint sed \
#   tmkms_bob \
#   -Ei 's/path = "\/root\/tmkms_bob\/secrets\/cosmoshub-3-consensus.key"/path = "\/root\/tmkms_bob\/secrets\/valBob_consensus.key"/g' \
#   /root/tmkms_bob/tmkms.toml

# echo "Update the path to the state file to represent the correct chain id"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#   --entrypoint sed \
#   tmkms_bob \
#   -i 's/state_file = "\/root\/tmkms_bob\/state\/cosmoshub-3-consensus.json"/state_file = "\/root\/tmkms_bob\/state\/priv_validator_key.json"/g' /root/tmkms_bob/tmkms.toml

# echo "Update chain id in tmkms.toml to layer"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#   --entrypoint sed \
#   tmkms_bob \
#   -i 's/id = "cosmoshub-3"/id = "layer"/g' /root/tmkms_bob/tmkms.toml

# docker run --rm -i \
#   -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#   --entrypoint sed \
#   tmkms_bob \
#   -i 's/chain_ids = \["cosmoshub-3"\]/chain_ids = \["layer"\]/g' /root/tmkms_bob/tmkms.toml

# echo "Copy public consensus key to valBob"
# docker run --rm -t \
#     -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
#     layerd_i \
#     comet show-validator \
#     | tr -d '\n' | tr -d '\r' \
#     > prod-sim/deskBob/config/pub_validator_key_valBob.json

# echo "Moving priv validator key off of valBob"
# cp prod-sim/valBob/config/priv_validator_key.json \
#   prod-sim/deskBob/config/priv_validator_key_valBob.json
# mv prod-sim/valBob/config/priv_validator_key.json \
#   prod-sim/kmsBob/secrets/priv_validator_key_valBob.json

# echo "import validator key into tmkms softsign feature"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#     -w /root/tmkms_bob \
#     tmkms_bob \
#     softsign import secrets/priv_validator_key_valBob.json \
#     secrets/valBob_consensus.key

# echo "copy over validator key from sentryBob where it is empty to remove bob's info"
# cp prod-sim/sentryBob/config/priv_validator_key.json prod-sim/valBob/config/

# echo "Set Port that will be used to communicate with val bob over private connection"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#     --entrypoint sed \
#     tmkms_bob \
#     -Ei 's/^addr = "tcp:.*$/addr = "tcp:\/\/valBob:26699"/g' /root/tmkms_bob/tmkms.toml

# echo "Update the key format in tmkms"
# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#     --entrypoint sed \
#     tmkms_bob \
#     -Ei 's/account_key_prefix = "cosmospub"/account_key_prefix = "tellor"/g' /root/tmkms_bob/tmkms.toml

# docker run --rm -i \
#     -v $(pwd)/prod-sim/kmsBob:/root/tmkms_bob \
#     --entrypoint sed \
#     tmkms_bob \
#     -Ei 's/consensus_key_prefix = "cosmosvalconspub"/consensus_key_prefix = "tellorvalconspub"/g' /root/tmkms_bob/tmkms.toml

# echo "Inform valBob of the port to listen for the tmkms"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/priv_validator_laddr = ""/priv_validator_laddr = "tcp:\/\/0.0.0.0:26699"/g' \
#   /root/.layer/valBob/config/config.toml

# echo "valBob config to not look for consensus key"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/^priv_validator_key_file/# priv_validator_key_file/g' \
#   /root/.layer/valBob/config/config.toml

# echo "Comment out validator state file so valBob no longer looks for it"
# docker run --rm -i \
#   -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
#   --entrypoint sed \
#   layerd_i \
#   -Ei 's/^priv_validator_state_file/# priv_validator_state_file/g' \
#   /root/.layer/valBob/config/config.toml

# cp prod-sim/sentryBob/config/priv_validator_key.json \
#     prod-sim/valBob/config/

# set chain id in genesis file on alice desktop
echo "set chain id in genesis file on Alice desktop"
docker run --rm -i \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    --entrypoint sed \
    layerd_i \
    -ie 's/"chain_id": .*"/"chain_id": '\"layer\"'/g' \
    /root/.layer/deskAlice/config/genesis.json


#Get the address returned from the keyring on alice desktop
echo "Set the address returned from the keyring on alice desktop"
ALICE=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskAlice \
    show valAlice --address)
echo $ALICE

# give loya to alice
echo "give loya to alice..."
docker run --rm -it \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskAlice \
    genesis add-genesis-account $ALICE 10000000000000loya 

# Update vote_extensions_enable_height in genesis.json
echo "Updating vote_extensions_enable_height in genesis.json..."
echo "Alice..."
jq '.consensus.params.abci.vote_extensions_enable_height = "1"' prod-sim/deskAlice/config/genesis.json > temp.json && mv temp.json prod-sim/deskAlice/config/genesis.json

#move genesis file from alice to bob desktop
echo "move genesis file from alice to bob desktop"
mv prod-sim/deskAlice/config/genesis.json \
    prod-sim/deskBob/config/

# Gets Bobs address from his desktop to be used to send loya to him
echo "Gets Bobs address from his desktop to be used to send loya to him"
BOB=$(echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskBob \
    show valBob --address)
echo $BOB

#send loya to bobs account
echo "send loya to bobs account"
docker run --rm -it \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskBob \
    genesis add-genesis-account $BOB 10000000000000loya 

# Copy validator keys from val nodes to bob and alice desktop
echo "Copy validator keys from val nodes to bob and alice desktop"
cp prod-sim/valAlice/config/priv_validator_key.json \
    prod-sim/deskAlice/config/priv_validator_key.json

cp prod-sim/valBob/config/priv_validator_key.json \
    prod-sim/deskBob/config/priv_validator_key.json

# Create gentx transaction for Bob to stake loya as validator
echo "Create gentx transaction for Bob to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    genesis gentx valBob 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskBob \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# move genesis file from bob to alice so that alice knows about bob
echo "move genesis file from bob to alice so that alice knows about bob"
mv prod-sim/deskBob/config/genesis.json \
    prod-sim/deskAlice/config/genesis.json

# create gentx tx for alice to stake loya as validator
echo "create gentx tx for alice to stake loya as validator..."
echo $PASSWORD | docker run --rm -i \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    genesis gentx valAlice 1000000000000loya \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskAlice \
    --chain-id layer 
    # --account-number 0 --sequence 0 \
    # --gas 1000000 \
    # --gas-prices 0.1loya

# copy over gentx transaction so that alice has both the gentx transactions then verify
echo "copy over gentx transaction so that alice has both the gentx transactions then verify"
cp prod-sim/deskBob/config/gentx/gentx-* \
    prod-sim/deskAlice/config/gentx

echo "Collection gentxs in desk alice"
docker run --rm -it \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    genesis collect-gentxs --home /root/.layer/deskAlice

# validate genesis file
echo "validate genesis file..."
docker run --rm -it \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    genesis validate-genesis --home /root/.layer/deskAlice

# ensure all nodes have the same genesis file
echo "ensure all nodes have the same genesis file...."
for name in deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
    cp prod-sim/deskAlice/config/genesis.json prod-sim/$name/config/genesis.json
done

# Get node info to be used in config values of other nodes
ALICE_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
    layerd_i \
    --home /root/.layer/valAlice \
    comet show-node-id)

ALICE_IDENTIFIER=$ALICE_VAL_NODE_ID@valAlice:26656
echo $ALICE_IDENTIFIER

ALICE_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer/sentryAlice \
    layerd_i \
    --home /root/.layer/sentryAlice \
    comet show-node-id)

ALICE_SENTRY_IDENTIFIER=$ALICE_SENTRY_NODE_ID@sentryAlice:26656
echo $ALICE_SENTRY_IDENTIFIER

BOB_VAL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
    layerd_i \
    --home /root/.layer/valBob \
    comet show-node-id)

BOB_IDENTIFIER=$BOB_VAL_NODE_ID@valBob:26656
echo $BOB_IDENTIFIER

BOB_SENTRY_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer/sentryBob \
    layerd_i \
    --home /root/.layer/sentryBob \
    comet show-node-id)

BOB_SENTRY_IDENTIFIER=$BOB_SENTRY_NODE_ID@sentryBob:26656
echo $BOB_SENTRY_IDENTIFIER


CAROL_NODE_ID=$(docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer/nodeCarol \
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
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer/sentryAlice \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_IDENTIFIER'"/g' /root/.layer/sentryAlice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer/sentryAlice \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$ALICE_SENTRY_SEEDS'"/g' /root/.layer/sentryAlice/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryAlice:/root/.layer/sentryAlice \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$ALICE_VAL_NODE_ID'"/g' /root/.layer/sentryAlice/config/config.toml


# Update sentryBob config.toml file
echo "Update sentryBob config.toml file"
docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer/sentryBob \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_IDENTIFIER'"/g' /root/.layer/sentryBob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer/sentryBob \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$BOB_SENTRY_SEEDS'"/g' /root/.layer/sentryBob/config/config.toml

docker run --rm -i \
    -v $(pwd)/prod-sim/sentryBob:/root/.layer/sentryBob \
    --entrypoint sed \
    layerd_i \
    -i 's/^private_peer_ids = ""/private_peer_ids = "'$BOB_VAL_NODE_ID'"/g' /root/.layer/sentryBob/config/config.toml

# Update valAlice config.toml file
echo "Update valAlice config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/valAlice:/root/.layer/valAlice \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$ALICE_SENTRY_IDENTIFIER'"/g' /root/.layer/valAlice/config/config.toml

#Update valBob config.toml file
echo "Update valBob config.toml file..."
docker run --rm -i \
    -v $(pwd)/prod-sim/valBob:/root/.layer/valBob \
    --entrypoint sed \
    layerd_i \
    -i 's/^persistent_peers = ""/persistent_peers = "'$BOB_SENTRY_IDENTIFIER'"/g' /root/.layer/valBob/config/config.toml

#Update nodeCarol config.toml file
echo "Update nodeCarol config.toml file.."
docker run --rm -i \
    -v $(pwd)/prod-sim/nodeCarol:/root/.layer/nodeCarol \
    --entrypoint sed \
    layerd_i \
    -i 's/^seeds = ""/seeds = "'$CAROL_NODE_SEEDS'"/g' /root/.layer/nodeCarol/config/config.toml

#Update cors_allowed_origin
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' \
    /root/.layer/$name/config/config.toml
done

#Update enabled-unsafe-cors to true
for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

for name in nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' \
    /root/.layer/$name/config/app.toml
done

# set timeout_commit or block time to 500ms
echo "Modifying timeout_commit in config.toml for alice..."
for name in deskAlice deskBob nodeCarol sentryAlice sentryBob valAlice valBob; do
    docker run --rm -i \
    -v $(pwd)/prod-sim/$name:/root/.layer/$name \
    --entrypoint sed \
    layerd_i \
    -Ei 's/timeout_commit = "5s"/timeout_commit = "500ms"/' /root/.layer/$name/config/config.toml
done

# rm ./prod-sim/valAlice/config/priv_validator_key.json
# mv ./prod-sim/sentryAlice/config/priv_validator_key.json ./prod-sim/valAlice/config/priv_validator_key.json
# mkdir ./prod-sim/valAlice/keys
# cp ./prod-sim/deskAlice/keys ./prod-sim/valAlice/keys

# rm prod-sim/valBob/priv_validator_key.json
# mv prod-sim/sentryBob/config/priv_validator_key.json prod-sim/valBob/config/priv_validator_key.json
# mkdir ./prod-sim/valBob/keys
# cp ./prod-sim/deskBob/keys ./prod-sim/valBob/keys

# Export alice key from os backend and import to test backend
echo "Exporting alice key... with password:"
docker run --rm -i \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskAlice \
    export valAlice > ./prod-sim/deskAlice/alice_keyfile

echo "Importing alice key to test backend..."
docker run --rm -i \
    -v $(pwd)/prod-sim/deskAlice:/root/.layer/deskAlice \
    layerd_i \
    keys \
    --keyring-backend test --home /root/.layer/deskAlice \
    import valAlice /root/.layer/deskAlice/alice_keyfile


# Export bob key from os backend and import to test backend
echo "Exporting bob key..."
docker run --rm -i \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    keys \
    --keyring-backend $KEYRING_BACKEND --home /root/.layer/deskBob \
    export valBob > ./prod-sim/deskBob/bob_keyfile

echo "Importing bob key to test backend..."
docker run --rm -i \
    -v $(pwd)/prod-sim/deskBob:/root/.layer/deskBob \
    layerd_i \
    keys \
    --keyring-backend test --home /root/.layer/deskBob \
    import valBob /root/.layer/deskBob/bob_keyfile

cp -r ./prod-sim/deskAlice/keyring-test/ ./prod-sim/valAlice/
cp -r ./prod-sim/deskBob/keyring-test/ ./prod-sim/valBob/
cp ./prod-sim/deskAlice/passphrase.txt ./prod-sim/valAlice/
cp ./prod-sim/deskBob/passphrase.txt ./prod-sim/valBob/

sleep 10

echo "Starting the chain in all containers..."
docker compose \
    --file ./prod-sim/docker-compose.yml \
    --project-name layer-test up



