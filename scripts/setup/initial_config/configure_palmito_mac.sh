#!/bin/bash

# Stop execution if any command fails
set -e

export LAYER_NODE_URL=https://node-palmito.tellorlayer.com/rpc/
export RPC_NODE_ID=ac7c10dc3de67c4394271c564671eeed4ac6f0e0
export KEYRING_BACKEND="test"
export PEERS="ac7c10dc3de67c4394271c564671eeed4ac6f0e0@34.229.148.107:26656,8d19cdf430e491d6d6106863c4c466b75a17088a@54.153.125.203:26656,c7b175a5bafb35176cdcba3027e764a0dbd0811c@34.219.95.82:26656,05105e8bb28e8c5ace1cecacefb8d4efb0338ec6@18.218.114.74:26656,705f6154c6c6aeb0ba36c8b53639a5daa1b186f6@3.80.39.230:26656,1f6522a346209ee99ecb4d3e897d9d97633ae146@3.101.138.30:26656,3822fa2eb0052b36360a7a6e285c18cc92e26215@175.41.188.192:26656"
export LAYER_HOME="/Users/$USER/.layer"
export STATE_SYNC_RPC="https://node-palmito.tellorlayer.com/rpc/"
export KEY_NAME="test"

#print an ascii art of the tellor logo
echo "--------------------------------"
echo "Welcome to the..."
echo "
████████╗███████╗██╗     ██╗      ██████╗ ██████╗ 
╚══██╔══╝██╔════╝██║     ██║     ██╔═══██╗██╔══██╗
   ██║   █████╗  ██║     ██║     ██║   ██║██████╔╝
   ██║   ██╔══╝  ██║     ██║     ██║   ██║██╔══██╗
   ██║   ███████╗███████╗███████╗╚██████╔╝██║  ██║
   ╚═╝   ╚══════╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝                                                                   
"
echo "Palmito Setup Script for Mac"
echo "--------------------------------"
echo "This is a guided setup script for the Tellor Palmito."
echo ""
echo "LAYER_NODE_URL: $LAYER_NODE_URL"
echo "RPC_NODE_ID: $RPC_NODE_ID"
echo "KEYRING_BACKEND: $KEYRING_BACKEND"
echo "PEERS: $PEERS"
echo "LAYER_HOME: $LAYER_HOME"
echo "STATE_SYNC_RPC: $STATE_SYNC_RPC"
echo ""
echo "--------------------------------"
while true; do
    read -p "Do you want to continue? (y/n): " continue_choice
    
    case "$continue_choice" in
      y|Y|yes|Yes|YES)
        break
        ;;
      n|N|no|No|NO)
        echo "Exiting..."
        exit 1
        ;;
      "")
        echo "Please enter y (yes) or n (no)."
        echo ""
        ;;
      *)
        echo "Please enter y (yes) or n (no)."
        echo ""
        ;;
    esac
done

# check if .layer directory exists
if [ -d "$LAYER_HOME" ]; then
    echo "--------------------------------"
    echo ""
    echo "Error: .layer directory already exists. This script is for new setups only."
    echo "If you want to reconfigure, please backup and remove the existing .layer directory first."
    echo "--------------------------------"
    exit 1
fi

# initialize layer directory
echo "Initializing layer directory..."
./layerd init layer --chain-id layertest-4

export STATE_SYNC_NODE_ID=$(./layerd status --node $STATE_SYNC_RPC | jq -r '.node_info.id')

echo "Change min gas price to 0loya in config files..."
sed -i '' 's/[0-9]\+stake/0loya/g' $LAYER_HOME/config/app.toml

echo "Set Chain Id to layer in client config file..."
sed -i '' 's/^chain-id = .*$/chain-id = "layertest-4"/g' $LAYER_HOME/config/client.toml

# Modify timeout_commit in config.toml for node
echo "Modifying timeout_commit in config.toml for node..."
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYER_HOME/config/config.toml

# Rate at which packets can be sent, in bytes/second
sed -i '' 's/^send_rate = .*/send_rate = 10240000/' $LAYER_HOME/config/config.toml

# Rate at which packets can be received, in bytes/second
sed -i '' 's/^recv_rate = .*/recv_rate = 10240000/' $LAYER_HOME/config/config.toml

# Check if user wants to open up node's API and RPC to outside traffic
while true; do
    echo "--------------------------------"
    echo ""
    echo "Do you want to open up node's API and RPC to outside traffic?"
    echo "(Optional. May require additional configuration in your firewall...)"
    echo "1) Yes, open up traffic to the ports"
    echo "2) No, keep default (localhost only)"
    echo "--------------------------------"
    read -p "Select an option [1-2]: " open_ports_choice

    case "$open_ports_choice" in
      1)
        echo "Opening up node's API to outside traffic..."
        sed -i '' 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' $LAYER_HOME/config/app.toml

        echo "Opening up node's RPC to outside traffic..."
        sed -i '' 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' $LAYER_HOME/config/config.toml
        break
        ;;
      2)
        echo "Leaving API and RPC bound to localhost only."
        break
        ;;
      "")
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
      *)
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
    esac
done

# Modify cors to accept *
echo "Modify cors to accept *"
sed -i '' 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $LAYER_HOME/config/config.toml

# Modify keyring-backend in client.toml for node
echo "Modifying keyring-backend in client.toml for node..."
sed -i '' 's/^keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' $LAYER_HOME/config/client.toml
# update for main dir as well. why is this needed?
sed -i '' 's/keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' $LAYER_HOME/config/client.toml

rm -f $LAYER_HOME/config/genesis.json
# get genesis file from running node's rpc
echo "Getting genesis from RPC....."
if ! curl -f "$LAYER_NODE_URL/genesis" | jq '.result.genesis' > $LAYER_HOME/config/genesis.json; then
    echo "Error: Failed to download genesis file from $LAYER_NODE_URL"
    exit 1
fi

# set initial seeds / peers
echo "Running Tellor node id: $RPC_NODE_ID. Configuring persistent peers..."
sed -i '' 's/persistent_peers = ""/persistent_peers = "'$PEERS'"/g' $LAYER_HOME/config/config.toml

# Check if user wants to create or import an account
while true; do
    echo "--------------------------------"
    echo ""
    echo "Do you want to create or import a wallet account?"
    echo "1) Create a new account"
    echo "2) Import an existing account"
    echo "3) No account creation please"
    echo "--------------------------------"
    read -p "Select an option [1-3]: " account_choice
    
    case "$account_choice" in
      1)
        echo "Creating a new account..."
        echo "Please enter a name for your account:"
        read -p "Account name: " KEY_NAME
        ./layerd keys add $KEY_NAME --keyring-backend test
        echo "--------------------------------"
        echo "Please save your mnemonic in a secure location!"
        read -p "Press Enter to when you have it written down..."
        echo "--------------------------------"
        break
        ;;
      2)
        echo "Importing an existing account..."
        echo "Please enter a name for your account:"
        read -p "Account name: " KEY_NAME
        ./layerd keys add $KEY_NAME --recover --keyring-backend test
        break
        ;;
      3)
        echo "Skipping account creation."
        break
        ;;
      *)
        echo "Invalid option. Please select 1, 2, or 3."
        echo ""
        ;;
    esac
done

# Check if user wants to configure state sync
while true; do
    echo "--------------------------------"
    echo ""
    echo "Do you want to configure state sync?"
    echo "1) Yes, configure state sync"
    echo "2) No, skip state sync configuration"
    echo "--------------------------------"
    read -p "Select an option [1-2]: " statesync_choice

    case "$statesync_choice" in
      1)
        echo "Configuring state sync..."
        
        # set statesync enable = true
        sed -i '' "s|^enable = .*|enable = true|" $LAYER_HOME/config/config.toml
        sed -i '' "s|^rpc_servers = .*|rpc_servers = \"$STATE_SYNC_RPC,$STATE_SYNC_RPC\"|" $LAYER_HOME/config/config.toml

        # get current height from state sync node
        CURRENT_HEIGHT=$(./layerd status --node $STATE_SYNC_RPC | jq -r '.sync_info.latest_block_height')

        # set configs so temporary node will start
        export TRUSTED_HEIGHT=$(($CURRENT_HEIGHT - 35500))
        sed -i '' "s|^trust_height = .*|trust_height = $TRUSTED_HEIGHT|" $LAYER_HOME/config/config.toml
        export TRUSTED_HASH=$(curl -s "$STATE_SYNC_RPC/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
        sed -i '' "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" $LAYER_HOME/config/config.toml
        
        # set chunk_request_timeout = "30s"
        sed -i '' "s|^chunk_request_timeout = .*|chunk_request_timeout = \"30s\"|" $LAYER_HOME/config/config.toml

        # set chunk_fetchers = "2"
        sed -i '' "s|^chunk_fetchers = .*|chunk_fetchers = \"2\"|" $LAYER_HOME/config/config.toml
        
        echo "State sync configuration complete!"
        break
        ;;
      2)
        echo "Skipping state sync configuration."
        break
        ;;
      "")
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
      *)
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
    esac
done

echo "Configuration Complete!"

# Check if user wants to start the node now
while true; do
    echo "--------------------------------"
    echo ""
    echo "Do you want to start the layer node now?"
    echo "1) Yes, start the node now"
    echo "2) No, I'll start it manually later"
    echo "--------------------------------"
    read -p "Select an option [1-2]: " start_node_choice

    case "$start_node_choice" in
      1)
        echo "Starting layer node..."
        echo "It can take a few hours to download the state and begin syncing."
        echo "Note: The node will run in the foreground. Press Ctrl+C to stop."
        echo "Starting in 3 seconds..."
        sleep 3
        ./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger
        break
        ;;
      2)
        echo "Node startup skipped."
        echo "To start the node later, run:"
        echo "./layerd start --home ~/.layer --keyring-backend test --key-name $KEY_NAME --api.enable --api.swagger"
        break
        ;;
      "")
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
      *)
        echo "Invalid option. Please select 1 or 2"
        echo ""
        ;;
    esac
done
