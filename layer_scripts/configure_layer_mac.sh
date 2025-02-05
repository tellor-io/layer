#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

# set variables in your .bashrc before starting this script!
source ~/.zshrc

export LAYER_NODE_URL=tellorlayer.com
export TELLORNODE_ID=5ca2c0eccb54e907ba474ce3b6827077ae40ba53
export KEYRING_BACKEND="test"
export PEERS="7f2c8cad741c28d7a01d9f1cf2e1a87eb751afa3@52.53.226.18:26656,f4786bc2a40172e29784b9f8d69567c474de8a8c@13.212.32.99:26656,59fd40b86c9b65ca717b29ce37b08fdb82c8e61d@18.224.20.250:26757,fc1caebd2550a4172bcdc073d0f18e630c44cc26@3.140.238.60:26757"

echo "Change denom to loya in config files..."
sed -i '' 's/([0-9]+)stake/1loya/g' ~/.layer/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i '' 's/^chain-id = .*$/chain-id = "layer"/g' ~/.layer/config/app.toml

# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for node..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' ~/.layer/config/config.toml

# Open up node to outside traffic
echo "Open up node to outside traffice" 
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
curl $LAYER_NODE_URL:26657/genesis | jq '.result.genesis' > ~/.layer/config/genesis.json

# set initial seeds / peers
echo "Running Tellor node id: $TELLORNODE_ID"
sed -i '' 's/seeds = ""/seeds = "'$PEERS'"/g' ~/.layer/config/config.toml
sed -i '' 's/persistent_peers = ""/persistent_peers = "'$PEERS'"/g' ~/.layer/config/config.toml


echo "layer has been configured in it's home folder!"
