#!/bin/bash
set -e

# Check if script is run with sudo privileges
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run with sudo privileges"
   echo "Please run: sudo $0"
   exit 1
fi

# Check if layer_snapshot service is already running
if sudo systemctl status layer_snapshot 2>&1 | grep -q 'Active: active (running)'; then
    echo "A snapshot process may already be running. Please stop it before using this script."
    exit 1
fi

# Function to check if a command was successful
check_error() {
    if [ $? -ne 0 ]; then
        echo "Error: $1"
        exit 1
    fi
}

# Function to wait for sync completion
wait_for_sync() {
    echo "Waiting for node to sync..."
    while true; do
        # Capture the last few lines of the journal in a variable
        log_output=$(journalctl --unit=layer_snapshot -n 40 --no-pager)

        if echo "$log_output" | grep -q "received complete proposal block"; then
            echo "Node sync completed!"
            break
        elif echo "$log_output" | grep -q "error from light block request from primary"; then
            echo "Error from light block request from primary (block not found)"
            echo "Stopping snapshot service due to light block request error..."
            sudo systemctl stop layer_snapshot
            check_error "Failed to stop layer_snapshot service"
            exit 1
        fi
        sleep 10
    done
}


echo "This script will remove the .layer_snapshot folder and create a new one."
echo "All the data in the .layer_snapshot folder will be lost."
read -p "Ready to configure the .layer_snapshot folder? [Y/n]: " init_data
init_data=${init_data:-Y}
echo "init_data: $init_data"
if [[ "$init_data" =~ ^[Yy]$ ]]; then
    # Remove existing .layer_snapshot folder
    sudo rm -rf /home/$(logname)/.layer_snapshot
    check_error "Failed to remove existing .layer_snapshot directory"

    echo "Initializing layer_snapshot home"
    #initialize layer_snapshot home
    sudo ./layerd init layer_snapshot --chain-id layertest-4 --home /home/$(logname)/.layer_snapshot
    check_error "Failed to initialize layer_snapshot"
    sudo chown -R $(logname):$(logname) /home/$(logname)/.layer_snapshot
    sudo chmod -R u+rwX /home/$(logname)/.layer_snapshot

    # add snapshot dummy account (do not use this account for anything real)
    sudo ./layerd keys add snapshot --keyring-backend test --home /home/$(logname)/.layer_snapshot
    check_error "Failed to add snapshot key"

    # remove initial config files from layer_snapshot home
    sudo rm /home/$(logname)/.layer_snapshot/config/app.toml  /home/$(logname)/.layer_snapshot/config/client.toml  /home/$(logname)/.layer_snapshot/config/config.toml  /home/$(logname)/.layer_snapshot/config/genesis.json
    check_error "Failed to remove old config files"

    # copy genesis from main home (.layer) to snapshot home (.layer_snapshot)
    echo "copying genesis from main home (.layer) to snapshot home (.layer_snapshot)"
    sudo cp /home/$(logname)/.layer/config/app.toml /home/$(logname)/.layer_snapshot/config/app.toml
    check_error "Failed to copy app.toml"
    echo "copying configs from main home(.layer) to snapshot home (.layer_snapshot)"
    sudo cp /home/$(logname)/.layer/config/client.toml /home/$(logname)/.layer_snapshot/config/client.toml
    check_error "Failed to copy client.toml"
    sudo cp /home/$(logname)/.layer/config/config.toml /home/$(logname)/.layer_snapshot/config/config.toml
    check_error "Failed to copy config.toml"
    sudo cp /home/$(logname)/.layer/config/genesis.json /home/$(logname)/.layer_snapshot/config/genesis.json
    check_error "Failed to copy genesis.json"

    echo "adding 101 to 2nd node's port numbers to avoid port conflicts with main node"
    update_ports() {
        file="$1"
        sudo sed -i \
            -e 's|address = "tcp://localhost:1317"|address = "tcp://localhost:1418"|g' \
            -e 's|address = "tcp://127.0.0.1:1317"|address = "tcp://127.0.0.1:1418"|g' \
            -e 's|address = "tcp://0.0.0.0:1317"|address = "tcp://0.0.0.0:1418"|g' \
            -e 's|address = "localhost:9090"|address = "localhost:9191"|g' \
            -e 's|address = "tcp://localhost:9090"|address = "tcp://localhost:9191"|g' \
            -e 's|address = "tcp://0.0.0.0:9090"|address = "tcp://0.0.0.0:9191"|g' \
            -e 's|address = "0.0.0.0:9090"|address = "0.0.0.0:9191"|g' \
            -e 's|node = "tcp://localhost:26657"|node = "tcp://localhost:26758"|g' \
            -e 's|proxy_app = "tcp://127.0.0.1:26658"|proxy_app = "tcp://127.0.0.1:26759"|g' \
            -e 's|laddr = "tcp://127.0.0.1:26656"|laddr = "tcp://127.0.0.1:26757"|g' \
            -e 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://127.0.0.1:26758"|g' \
            -e 's|pprof_laddr = "localhost:6060"|pprof_laddr = "localhost:6161"|g' \
            -e 's|laddr = "tcp://0.0.0.0:26656"|laddr = "tcp://0.0.0.0:26757"|g' \
            -e 's|laddr = "tcp://0.0.0.0:26657"|laddr = "tcp://0.0.0.0:26758"|g' \
            "$file"
        check_error "Failed to update ports in $file"
    }

    # Update ports in the copied configuration files
    echo "Updating ports..."
    update_ports /home/$(logname)/.layer_snapshot/config/app.toml
    update_ports /home/$(logname)/.layer_snapshot/config/client.toml
    update_ports /home/$(logname)/.layer_snapshot/config/config.toml

    # create statesync configuration for snapshot node
    sudo sed -i 's|^rpc_servers = .*|rpc_servers = "http://localhost:26657,http://localhost:26657"|' /home/$(logname)/.layer_snapshot/config/config.toml

    echo "adding the main node's rpc as the snapshot node's persistent peer"
    sudo ./layerd status 

    PRIMARY_NODE_ID=$(sudo ./layerd status | jq -r '.node_info.id')

    sudo sed -i 's/^seeds = ".*"/seeds = ""/' /home/$(logname)/.layer_snapshot/config/config.toml
    sudo sed -i "s/^persistent_peers = \".*\"/persistent_peers = \"${PRIMARY_NODE_ID}@127.0.0.1:26656\"/" /home/$(logname)/.layer_snapshot/config/config.toml

    echo "Modify cors to accept *"
    sudo sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["*"\]/g' /home/$(logname)/.layer/config/config.toml

    # set state sync to true
    sudo sed -i 's/^enable = .*/enable = true/' /home/$(logname)/.layer_snapshot/config/config.toml

    # set state sync to false on main node just in case
    sudo sed -i 's/^enable = .*/enable = false/' /home/$(logname)/.layer/config/config.toml

    # Rate at which packets can be sent, in bytes/second
    sudo sed -i 's/^send_rate = .*/send_rate = 10240000/' /home/$(logname)/.layer/config/config.toml
    sudo sed -i 's/^send_rate = .*/send_rate = 10240000/' /home/$(logname)/.layer_snapshot/config/config.toml

    # Rate at which packets can be received, in bytes/second
    sudo sed -i 's/^recv_rate = .*/recv_rate = 10240000/' /home/$(logname)/.layer/config/config.toml
    sudo sed -i 's/^recv_rate = .*/recv_rate = 10240000/' /home/$(logname)/.layer_snapshot/config/config.toml

    echo "configured values in /home/$(logname)/.layer/config/config.toml:"
    sudo grep -E -C 1 'address = |laddr = |pprof_laddr =|enable =|cors_allowed_origins =|rpc_servers = |seeds = |persistent_peers = |send_rate = |recv_rate = ' /home/$(logname)/.layer/config/config.toml

    echo "configured values in /home/$(logname)/.layer_snapshot/config/config.toml:"
    sudo grep -E -C 1 'address = |laddr = |pprof_laddr =|enable =|cors_allowed_origins =|rpc_servers = |seeds = |persistent_peers = |send_rate = |recv_rate = ' /home/$(logname)/.layer_snapshot/config/config.toml
fi

# Remove old chain data
if [ -f "/home/$(logname)/.layer_snapshot/data/blockstore.db" ]; then
    read -p "Do you want to remove old snapshot chain data for retries? [y/N]: " remove_data
    remove_data=${remove_data:-N}
    if [[ "$remove_data" =~ ^[Yy]$ ]]; then
        echo "Removing old snapshot chain data for retries..."
        sudo rm -rf /home/$(logname)/.layer_snapshot/data/application.db \
               /home/$(logname)/.layer_snapshot/data/blockstore.db \
               /home/$(logname)/.layer_snapshot/data/cs.wal \
               /home/$(logname)/.layer_snapshot/data/evidence.db \
               /home/$(logname)/.layer_snapshot/data/snapshots \
               /home/$(logname)/.layer_snapshot/data/state.db \
               /home/$(logname)/.layer_snapshot/data/tx_index.db
        check_error "Failed to remove old chain data"
    else
        echo "Skipping removal of old snapshot chain data."
    fi
fi

# Get trusted height and hash
# Get the highest snapshot height from the snapshots directory
SNAPSHOT_DIR="/home/$(logname)/.layer/data/snapshots"
SNAPSHOT_HEIGHTS=($(ls "$SNAPSHOT_DIR" | grep -E '^[0-9]+$' | sort -n))

if [ ${#SNAPSHOT_HEIGHTS[@]} -eq 0 ]; then
    echo "Error: No snapshot heights found in directory"
    exit 1
fi

echo "Available snapshot heights:"
for i in "${!SNAPSHOT_HEIGHTS[@]}"; do
    echo "$((i+1)). ${SNAPSHOT_HEIGHTS[$i]}"
done
echo "$(( ${#SNAPSHOT_HEIGHTS[@]} + 1 )). Enter a custom height"
echo "$(( ${#SNAPSHOT_HEIGHTS[@]} + 2 )). Offset current block height -8000"

while true; do
    read -p "Select a snapshot height by number (1-${#SNAPSHOT_HEIGHTS[@]}) or enter $(( ${#SNAPSHOT_HEIGHTS[@]} + 1 )) for manual input, or $(( ${#SNAPSHOT_HEIGHTS[@]} + 2 )) for offset: " choice
    if [[ "$choice" =~ ^[0-9]+$ ]]; then
        if [ "$choice" -ge 1 ] && [ "$choice" -le "${#SNAPSHOT_HEIGHTS[@]}" ]; then
            SNAPSHOT_HEIGHT="${SNAPSHOT_HEIGHTS[$((choice-1))]}"
            TRUSTED_HEIGHT=$SNAPSHOT_HEIGHT
            TRUSTED_HASH=$(curl -s "http://localhost:26657/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
            check_error "Failed to get trusted hash"
            break
        elif [ "$choice" -eq $(( ${#SNAPSHOT_HEIGHTS[@]} + 1 )) ]; then
            read -p "Enter the trusted height to use: " manual_height
            if [[ "$manual_height" =~ ^[0-9]+$ ]]; then
                TRUSTED_HEIGHT="$manual_height"
                TRUSTED_HASH=$(curl -s "http://localhost:26657/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
                check_error "Failed to get trusted hash"
                break
            else
                echo "Invalid manual height. Please enter a number."
            fi
        elif [ "$choice" -eq $(( ${#SNAPSHOT_HEIGHTS[@]} + 2 )) ]; then
            export LATEST_HEIGHT=$(curl -s http://127.0.0.1:26657/block | jq -r .result.block.header.height)
            export TRUSTED_HEIGHT=$((LATEST_HEIGHT-8000))
            export TRUSTED_HASH=$(curl -s "http://127.0.0.1:26657/block?height=$TRUSTED_HEIGHT" | jq -r .result.block_id.hash)
            echo $TRUSTED_HEIGHT $TRUSTED_HASH
            break
        else
            echo "Invalid selection. Please try again."
        fi
    else
        echo "Invalid selection. Please try again."
    fi
done

echo "Selected trusted height: $TRUSTED_HEIGHT"
echo "Trusted hash: $TRUSTED_HASH"

# Update trust_height and trust_hash
echo "Updating trust_height and trust_hash..."
sudo sed -i "s|^trust_height = .*|trust_height = $TRUSTED_HEIGHT|" /home/$(logname)/.layer_snapshot/config/config.toml
sudo sed -i "s|^trust_hash = .*|trust_hash = \"$TRUSTED_HASH\"|" /home/$(logname)/.layer_snapshot/config/config.toml

echo "Starting layer_snapshot service"
sudo systemctl start layer_snapshot
echo "use sudo journalctl -fu layer_snapshot in another terminal to monitor status of the snapshot node"

wait_for_sync

# echo "Do you want to stop the main node and" 
# echo "replace the data in /home/$(logname)/.layer/data with the data from /home/$(logname)/.layer_snapshot/data?"
# read -p "Note: the priv_validator_state.json file is preserved in /home/$(logname)/.layer/data [y/N]: " replace_data
# replace_data=${replace_data:-N}
# if [[ ! "$replace_data" =~ ^[Yy]$ ]]; then
#     echo "Exiting without replacing /home/$(logname)/.layer/data folder!"
#     exit 0
# fi

sudo systemctl stop layer
sudo systemctl stop layer_snapshot
rm -rf /home/$(logname)/.layer/data/application.db
rm -rf /home/$(logname)/.layer/data/blockstore.db
rm -rf /home/$(logname)/.layer/data/cs.wal
rm -rf /home/$(logname)/.layer/data/evidence.db
rm -rf /home/$(logname)/.layer/data/snapshots
rm -rf /home/$(logname)/.layer/data/state.db
rm -rf /home/$(logname)/.layer/data/tx_index.db

mv /home/$(logname)/.layer_snapshot/data/application.db /home/$(logname)/.layer/data/application.db
mv /home/$(logname)/.layer_snapshot/data/blockstore.db /home/$(logname)/.layer/data/blockstore.db
mv /home/$(logname)/.layer_snapshot/data/cs.wal /home/$(logname)/.layer/data/cs.wal
mv /home/$(logname)/.layer_snapshot/data/evidence.db /home/$(logname)/.layer/data/evidence.db
mv /home/$(logname)/.layer_snapshot/data/snapshots /home/$(logname)/.layer/data/snapshots
mv /home/$(logname)/.layer_snapshot/data/state.db /home/$(logname)/.layer/data/state.db
mv /home/$(logname)/.layer_snapshot/data/tx_index.db /home/$(logname)/.layer/data/tx_index.db

# Fix ownership so the user can access the files
sudo chown -R $(logname):$(logname) /home/$(logname)/.layer_snapshot
sudo chown -R $(logname):$(logname) /home/$(logname)/.layer

sudo systemctl start layer

echo "Storage reset process is complete!"
echo "use sudo journalctl -fu layer to check for success"
