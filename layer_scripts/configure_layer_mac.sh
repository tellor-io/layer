#!/bin/zsh

# clear the terminal
clear

# Stop execution if any command fails
set -e

# set variables in your .bashrc before starting this script!
source ~/.zshrc

export LAYER_NODE_URL=https://node-palmito.tellorlayer.com/rpc/
export TELLORNODE_ID=ac7c10dc3de67c4394271c564671eeed4ac6f0e0
export KEYRING_BACKEND="test"
export PEERS="ac7c10dc3de67c4394271c564671eeed4ac6f0e0@34.229.148.107:26656,c7b175a5bafb35176cdcba3027e764a0dbd0811c@34.219.95.82:26656,05105e8bb28e8c5ace1cecacefb8d4efb0338ec6@18.218.114.74:26656,8d19cdf430e491d6d6106863c4c466b75a17088a@54.153.125.203:26656"

echo "Change denom to loya in config files..."
sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i '' 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/config/app.toml

# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for node..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffic"
sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26656"/laddr = "tcp:\/\/0.0.0.0:26656"/g' ~/.layer/config/config.toml

sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' ~/.layer/config/app.toml

# Modify cors to accept *
echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/config.toml

# enable unsafe cors
echo "Enable unsafe cors"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' ~/.layer/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i '' 's/^enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' ~/.layer/config/app.toml
sed -i '' 's/^enable-unsafe-cors = false/enable-unsafe-cors = true/g' ~/.layer/config/app.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for node..."
sed -i '' 's/^keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' ~/.layer/config/client.toml

rm -f ~/.layer/config/genesis.json
# get genesis file from running node's rpc
echo "Getting genesis from runnning node....."
curl $LAYER_NODE_URL/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json

# set initial seeds / peers
echo "Running Tellor node id: $TELLORNODE_ID"
sed -i '' 's/seeds = ""/seeds = "'$PEERS'"/g' ~/.layer/config/config.toml
sed -i '' 's/persistent_peers = ""/persistent_peers = "'$PEERS'"/g' ~/.layer/config/config.toml


echo "layer has been configured in ~/.layer !"