#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

## YOU WILL NEED TO SET THIS TO WHATEVER NODE YOU WOULD LIKE TO USE
export LAYER_NODE_URL=54.166.101.67
export TELLORNODE_ID=72a0284c589e1e11823c27580bfbcbaa32a769e7
export KEYRING_BACKEND=test
export NODE_MONIKER="reportermoniker"
export NODE_NAME="reporter"
export LAYERD_NODE_HOME="$HOME/.layer/$NODE_NAME"


# Remove old test chain data (if present)
echo "Removing old test chain data..."
rm -rf ~/.layer
rm -rf ~/.layer/$NODE_NAME

# Build layerd
echo "Building layerd..."
go build ./cmd/layerd

# Initialize the chain
echo "Initializing the chain..."
./layerd init layer --chain-id layer

# Initialize chain node with the folder for alice
echo "Initializing chain node for alice..."
./layerd init $NODE_MONIKER --chain-id layer --home ~/.layer/$NODE_NAME

echo "creating keys for node"
./layerd keys add $NODE_NAME --home ~/.layer/$NODE_NAME --keyring-backend $KEYRING_BACKEND

# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for $NODE_NAME..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/$NODE_NAME/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffic" 
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/$NODE_NAME/config/config.toml

sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/$NODE_NAME/config/app.toml

# Modify cors to accept *
echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/config.toml

# enable unsafe cors
echo "Enable unsafe cors"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/$NODE_NAME/config/app.toml
sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for node..."
sed -i '' 's/^keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/$NODE_NAME/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml

rm -f ~/.layer/config/genesis.json
rm -f ~/.layer/$NODE_NAME/config/genesis.json
# get genesis file from running node's rpc
echo "Getting genesis from runnning node....."
curl $LAYER_NODE_URL:26657/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json
curl $LAYER_NODE_URL:26657/genesis | jq '.result.genesis' > ~/.layer/$NODE_NAME/config/genesis.json

echo "Running Tellor node id: $TELLORNODE_ID"
sed -i '' 's/seeds = ""/seeds = "'$TELLORNODE_ID'@'$LAYER_NODE_URL':26656"/g' ~/.layer/$NODE_NAME/config/config.toml
sed -i '' 's/persistent_peers = ""/persistent_peers = "'$TELLORNODE_ID'@'$LAYER_NODE_URL':26656"/g' ~/.layer/$NODE_NAME/config/config.toml
sleep 10

echo "Starting chain for node..."
# ./layerd start --home $LAYERD_NODE_HOME --api.enable --api.swagger --panic-on-daemon-failure-enabled=false --p2p.seeds "$TELLORNODE_ID@$LAYER_NODE_URL:26656"
./layerd start --home $LAYERD_NODE_HOME --api.swagger --price-daemon-enabled=false --p2p.seeds "$TELLORNODE_ID@$LAYER_NODE_URL:26656"


# validator address: tellorvaloper1a6dhndqrw9xx6ylt6gc6mkl0uhwugkw6r409l3
#./layerd tx staking delegate tellorvaloper1a6dhndqrw9xx6ylt6gc6mkl0uhwugkw6r409l3 10000000loya --gas auto --keyring-backend test --home ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l --node=http://tellornode.com:26657 --chain-id layer
#./layerd tx reporter create-reporter 200 1000000 --gas auto --keyring-backend test --keyring-dir ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l --node=http://tellornode.com:26657 --chain-id layer

# ./layerd tx oracle submit-value tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000aa36a700000000000000000000000088df592f8eb5d7bd38bfef7deb0fbc02cf3778a00000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000418160ddd00000000000000000000000000000000000000000000000000000000 000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000190c2aa29b1000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000002322afa7f323edc40ff64 0x00 --gas auto --keyring-backend test --keyring-dir ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l --node=http://tellornode.com:26657 --chain-id layer

./layerd tx registry register-spec EVMCall {"document_hash": "evm call data spec","response_value_type": "bytes","abi_components": \\[{"name": "chainId","field_type": "uint256"}, {"name": "contractAddress","field_type": "address"}, {"name": "calldata","field_type": "bytes"}],"aggregation_mothod": "mode","registrar": "tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l","report_buffer_window": "300s"} --keyring-backend test --keyring-dir ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l

{
    document_hash: string,
    response_value_type string,
    abi_components { name: string, field_type: string, }
    aggregation_mothod: string,
    registrar: address string,
    report_buffer_window: time duration (try 1s)
}

DATA_SPEC_JSON=$(cat <<EOF
{
    "document_hash": "evm call data spec",
    "response_value_type": "bytes",
    "abi_components": [{
        "name": "chainId",
        "field_type": "uint256",
    }, {
        "name": "contractAddress",
        "field_type": "address",
    }, {
        "name": "calldata",
        "field_type": "bytes",
    }],
    "aggregation_mothod": "WeightedMode",
    "registrar": "tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l",
    "report_buffer_window": "300s",
}
EOF
)


{"document_hash": "evm call data spec","response_value_type": "bytes","abi_components": [{"name": "chainId","field_type": "uint256",}, {"name": "contractAddress","field_type": "address",}, {"name": "calldata","field_type": "bytes",}],"aggregation_mothod": "mode","registrar": "tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l","report_buffer_window": "300s",}
EOF
)

echo "$DATA_SPEC_JSON" > ./data_spec_json.json


./layerd tx oracle tip tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000aa36a700000000000000000000000088df592f8eb5d7bd38bfef7deb0fbc02cf3778a00000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000418160ddd00000000000000000000000000000000000000000000000000000000 50000loya --keyring-backend test --keyring-dir ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l --chain-id layer --node=http://tellornode.com:26657