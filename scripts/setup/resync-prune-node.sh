#!/bin/bash

# Check if script is run with sudo (it should be run with sudo.)
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script should be run with sudo."
    echo "Please run as sudo: sudo $0 --network <tellor-1|layertest-4>"
    exit 1
fi

# Stop execution if any command fails
set -e

# Detect operating system. error out if not linux.
if [ "$(uname -s)" != "Linux" ]; then
    echo "Error: This script is for linux only."
    exit 1
fi

# Initialize variables
SNAPSHOT_PATH=""
NETWORK=""
DISCORD_WEBHOOK=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --network|-n)
            NETWORK="$2"
            shift 2
            ;;
        --snapshot|-s)
            SNAPSHOT_PATH="$2"
            shift 2
            ;;
        --discord-webhook|-d)
            DISCORD_WEBHOOK="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: sudo $0 --network <tellor-1|layertest-4> [--snapshot <path>] [--discord-webhook <url>]"
            echo ""
            echo "Options:"
            echo "  --network, -n         Required. Network to use: mainnet or palmito"
            echo "  --snapshot, -s        Optional. Path to pre-downloaded snapshot file"
            echo "  --discord-webhook, -d Optional. Discord webhook URL for notifications"
            echo "  --help, -h            Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: sudo $0 --network <tellor-1|layertest-4> [--snapshot <path>] [--discord-webhook <url>]"
            exit 1
            ;;
    esac
done

# Validate NETWORK flag is provided
if [ -z "$NETWORK" ]; then
    echo "Error: --network flag is required."
    echo "Usage: sudo $0 --network <tellor-1|layertest-4> [--snapshot <path>] [--discord-webhook <url>]"
    exit 1
fi

# Validate NETWORK value
if [ "$NETWORK" != "tellor-1" ] && [ "$NETWORK" != "layertest-4" ]; then
    echo "Error: Invalid network '$NETWORK'. Must be 'tellor-1' or 'layertest-4'."
    echo "Usage: sudo $0 --network <tellor-1|layertest-4> [--snapshot <path>] [--discord-webhook <url>]"
    exit 1
fi

# Fix 14: Use SUDO_USER as fallback for logname (for cron/systemd contexts)
ACTUAL_USER="${SUDO_USER:-$(logname)}"
USER_HOME="/home/$ACTUAL_USER"

# Function to send Discord alerts via webhook
discord_alert() {
    local message="$1"
    
    if [ -z "$DISCORD_WEBHOOK" ]; then
        return 0
    fi
    
    # Create JSON payload for Discord
    local json_payload=$(cat <<EOF
{
  "content": "**Tellor Layer Node Resync Alert**",
  "embeds": [{
    "title": "Node Resync Status",
    "description": "$message",
    "color": 5814783,
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)",
    "footer": {
      "text": "Network: $NETWORK | Chain ID: $CHAIN_ID"
    }
  }]
}
EOF
)
    
    # Send to Discord webhook and capture HTTP status code
    # Using -w to append HTTP code, -s for silent, -o to discard response body
    local http_code=$(curl -s -w "%{http_code}" -o /dev/null -X POST "$DISCORD_WEBHOOK" \
        -H "Content-Type: application/json" \
        -d "$json_payload")
    
    # Check if HTTP status code indicates success (2xx)
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo "✓ Discord alert sent successfully (HTTP $http_code)"
        return 0
    else
        echo "✗ Failed to send Discord alert (HTTP $http_code)"
        echo "  Check your webhook URL is correct"
        # Don't return error to avoid stopping script with set -e
        return 0
    fi
}

# Define version tags
LAYERD_TAG_MAINNET="v6.0.0"
LAYERD_TAG_PALMITO="v6.0.0"

# Set network-specific variables
if [ "$NETWORK" == "tellor-1" ]; then
    LAYERD_TAG="$LAYERD_TAG_MAINNET"
    PEERS=""
    LAYER_HOME="$USER_HOME/.layer"
    LAYER_SNAPSHOT_HOME="$USER_HOME/.layer_snapshot"
    CHAIN_ID="tellor-1"
elif [ "$NETWORK" == "layertest-4" ]; then
    LAYERD_TAG="$LAYERD_TAG_PALMITO"
    PEERS=""
    LAYER_HOME="$USER_HOME/.layer"
    LAYER_SNAPSHOT_HOME="$USER_HOME/.layer_snapshot"
    CHAIN_ID="layertest-4"
fi

# check if layer home directory exists. If it does, ask if they want to remove it before continuing.
if [ -d "$LAYER_SNAPSHOT_HOME" ]; then
    echo "Layer snapshot home directory found at $LAYER_SNAPSHOT_HOME."
    echo "It needs to be removed before continuing."
    echo "Press enter to continue..."
    read -p ""
    rm -rf $LAYER_SNAPSHOT_HOME
fi

echo "--------------------------------"
echo "NETWORK: $NETWORK"
echo "CHAIN_ID: $CHAIN_ID"
echo "LAYER_HOME: $LAYER_HOME"
echo "LAYER_SNAPSHOT_HOME: $LAYER_SNAPSHOT_HOME"
echo "LAYERD_TAG: $LAYERD_TAG"
echo "PEERS: $PEERS"
echo "--------------------------------"

# check if systemd service is configured as expected.
if [ ! -f "/etc/systemd/system/layer.service" ]; then
    echo "Error: Systemd service file not found at /etc/systemd/system/layer.service."
    echo "This script will only work if the service is configured as expected."
    exit 1
fi

# check if layer home directory exists. If it does not exist, error out.
if [ ! -d "$LAYER_HOME" ]; then
    echo "Error: Layer home directory not found at $LAYER_HOME."
    echo "This script will only work if the layer home directory exists."
    exit 1
fi

# init variables for mainnet and palmito
LAYERD_PATH="$USER_HOME/layer/binaries/$LAYERD_TAG/layerd"

# Get node status and verify it matches selected network
# Read the RPC address from client.toml config
CLIENT_TOML="$LAYER_HOME/config/client.toml"
if [ -f "$CLIENT_TOML" ]; then
    # NODE_ADDR_TCP for layerd CLI commands (tcp://...)
    NODE_ADDR_TCP=$(grep '^node = ' "$CLIENT_TOML" | sed 's/node = "\(.*\)"/\1/')
    # NODE_ADDR_HTTP for curl requests (http://...)
    NODE_ADDR_HTTP=$(echo "$NODE_ADDR_TCP" | sed 's|^tcp://|http://|')
fi
if [ -z "$NODE_ADDR_TCP" ]; then
    echo "Error: NODE_ADDR is not set. Check $CLIENT_TOML"
    exit 1
fi

echo "Checking $LAYERD_PATH status (node: $NODE_ADDR_TCP)..."
LOCAL_STATUS=$($LAYERD_PATH status --home $LAYER_HOME --node "$NODE_ADDR_TCP" 2>&1)
if [ $? -ne 0 ]; then
    echo "Error: Failed to get node status"
    echo "$LOCAL_STATUS"
    exit 1
fi
LOCAL_NODE_ID=$(echo "$LOCAL_STATUS" | jq -r '.node_info.id')
LOCAL_CHAIN_ID=$(echo "$LOCAL_STATUS" | jq -r '.node_info.network')
LOCAL_SYNC_STATUS=$(echo "$LOCAL_STATUS" | jq -r '.sync_info.catching_up')
LOCAL_LISTEN_ADDR=$(echo "$LOCAL_STATUS" | jq -r '.node_info.listen_addr' | sed 's|^tcp://||')
LOCAL_RPC_ADDR=$(echo "$LOCAL_STATUS" | jq -r '.node_info.other.rpc_address' | sed 's|^tcp://||')

if [ "$LOCAL_SYNC_STATUS" == "true" ]; then
    echo "Error: Local node is catching up."
    echo "It probably doesn't make sense to reset the chain data storage at this time."
    exit 1
fi
if [ "$LOCAL_CHAIN_ID" != "$CHAIN_ID" ]; then
    echo "Error: Local node chain ID does not match selected network."
    echo "Selected network: $NETWORK (chain ID: $CHAIN_ID)"
    echo "Local node chain ID: $LOCAL_CHAIN_ID"
    exit 1
fi

# set local node peer configuration.
LOCAL_NODE_PEER_CONFIG="$LOCAL_NODE_ID@$LOCAL_LISTEN_ADDR"

echo "--------------------------------"
echo "LOCAL_NODE_ID: $LOCAL_NODE_ID"
echo "LOCAL_CHAIN_ID: $LOCAL_CHAIN_ID"
echo "LOCAL_SYNC_STATUS: $LOCAL_SYNC_STATUS"
echo "LOCAL_LISTEN_ADDR: $LOCAL_LISTEN_ADDR"
echo "LOCAL_RPC_ADDR: $LOCAL_RPC_ADDR"
echo "LOCAL_NODE_PEER_CONFIG: $LOCAL_NODE_PEER_CONFIG"
echo "--------------------------------"


# print confirmation message
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
echo "Chain Data Resetter for Linux"
echo "--------------------------------"
echo "This is a quick-installer for Tellor Layer."
echo "This script will: "
echo "  1) Configure a 2nd layer node on your computer."
echo "  2) Download the latest snapshot from layer-node.com."
echo "  3) Extract the snapshot to $LAYER_SNAPSHOT_HOME."
echo "  4) Start the snapshot service waiting for the 2nd node to sync."
echo "  5) Stop the main node and the snapshot node."
echo "  6) Replace the chaindata in $LAYER_HOME/data"
echo "     with the data from $LAYER_SNAPSHOT_HOME/data and start the main node again."
echo "  7) Start the main node back up and confirm nominal operation."
echo "  8) Clean up."


while true; do
    read -p "Make these changes to your computer? (y/n): " continue_choice
    
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

echo "sending discord alert if webhook is configured..."
if [ -n "$DISCORD_WEBHOOK" ]; then
    echo "Sending discord alert..."
    discord_alert "Starting chain data resetter for $ACTUAL_USER on $NETWORK..."
    echo "--------------------------------"
else
    echo "No Discord webhook configured, skipping alert"
    echo "--------------------------------"
fi

echo ""
echo "================================"
echo " VERIFYING SNAPSHOT SERVICE..."
echo "================================"
echo ""
sleep 1

# create it if it's not already created.
if [ ! -f "/etc/systemd/system/layer_snapshot.service" ]; then
    echo "Creating layer_snapshot.service for $LAYER_SNAPSHOT_HOME..."
    echo "press enter to continue..."
    read -p ""
    cat > "/etc/systemd/system/layer_snapshot.service" << EOF
[Unit]
Description=Layer Snapshoting Service
After=network-online.target

[Service]
User=$ACTUAL_USER
Group=$ACTUAL_USER
WorkingDirectory=$LAYER_SNAPSHOT_HOME
ExecStart=$LAYERD_PATH start --home $LAYER_SNAPSHOT_HOME
Restart=always
RestartSec=10
MemoryMax=10G

[Install]
WantedBy=multi-user.target
EOF
    echo "layer_snapshot.service created successfully."
    # Fix 15: Reload systemd after creating new service
    echo "Reloading systemd daemon..."
    systemctl daemon-reload
fi


echo ""
echo "================================"
echo "  CHECKING BINARIES..."
echo "================================"
echo ""
sleep 1

# Check if binary already exists and verify version
LAYERD_BINARY_PATH="$USER_HOME/layer/binaries/$LAYERD_TAG/layerd"
if [ -f "$LAYERD_BINARY_PATH" ]; then
    echo "Binary found at $LAYERD_BINARY_PATH. Checking version..."
    EXISTING_VERSION=$($LAYERD_BINARY_PATH version 2>&1 | tr -d '\n')
    # Normalize versions by removing 'v' prefix for comparison
    NORMALIZED_EXISTING="${EXISTING_VERSION#v}"
    NORMALIZED_REQUIRED="${LAYERD_TAG#v}"
    echo "Comparing versions: existing=$NORMALIZED_EXISTING, required=$NORMALIZED_REQUIRED"
    if [ "$NORMALIZED_EXISTING" == "$NORMALIZED_REQUIRED" ]; then
        echo "✓ Binary version matches required version ($LAYERD_TAG)."
    else
        echo "Error: Binary version mismatch."
        echo "Expected version: $LAYERD_TAG"
        echo "Existing version: $EXISTING_VERSION"
        exit 1
    fi
else
    echo "Error: layerd binary not found at expected location."
    echo "Expected location: $LAYERD_BINARY_PATH"
    exit 1
fi

LAYERD_PATH="$USER_HOME/layer/binaries/$LAYERD_TAG/layerd"


# Fix 8: Check available disk space before proceeding
echo ""
echo "================================"
echo "  CHECKING DISK SPACE..."
echo "================================"
echo ""
sleep 1

# Get available space in GB for the user's home directory
AVAILABLE_SPACE_KB=$(df -k "$USER_HOME" | tail -1 | awk '{print $4}')
AVAILABLE_SPACE_GB=$((AVAILABLE_SPACE_KB / 1024 / 1024))

# Require at least 150GB free space (snapshot ~40-80GB + extraction space + safety margin)
REQUIRED_SPACE_GB=200

echo "Available disk space in $USER_HOME: ${AVAILABLE_SPACE_GB} GB"
echo "Required disk space: ${REQUIRED_SPACE_GB} GB"

if [ $AVAILABLE_SPACE_GB -lt $REQUIRED_SPACE_GB ]; then
    echo ""
    echo "ERROR: Insufficient disk space!"
    echo "You have ${AVAILABLE_SPACE_GB} GB available, but need at least ${REQUIRED_SPACE_GB} GB."
    echo "Please free up disk space before running this script."
    exit 1
fi

echo "✓ Sufficient disk space available"
echo ""

# initialize layer_snapshot directory
echo "Initializing layer_snapshot directory..."
echo "Running: $LAYERD_PATH init layer_snapshot --chain-id $CHAIN_ID --home $LAYER_SNAPSHOT_HOME"
sudo -u $ACTUAL_USER $LAYERD_PATH init layer_snapshot --chain-id $CHAIN_ID --home $LAYER_SNAPSHOT_HOME


echo ""
echo "======================================="
echo " INITIALIZING SNAPSHOT NODE CONFIGS..."
echo "======================================="
echo ""
sleep 1

# change denom, chain id, and timeout commit in config files
echo "Changing configs for $NETWORK..."
sed -i 's/[0-9]\+stake/0loya/g' $LAYER_SNAPSHOT_HOME/config/app.toml
sed -i 's/^chain-id = .*$/chain-id = "'$CHAIN_ID'"/g' $LAYER_SNAPSHOT_HOME/config/client.toml
sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYER_SNAPSHOT_HOME/config/config.toml
sed -i 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $LAYER_SNAPSHOT_HOME/config/config.toml
sed -i 's/^keyring-backend = "os"/keyring-backend = "test"/g' $LAYER_SNAPSHOT_HOME/config/client.toml
sed -i 's/persistent_peers = ""/persistent_peers = "'$PEERS'"/g' $LAYER_SNAPSHOT_HOME/config/config.toml
sed -i 's/^send_rate = .*/send_rate = 10240000/' $LAYER_SNAPSHOT_HOME/config/config.toml
sed -i 's/^recv_rate = .*/recv_rate = 10240000/' $LAYER_SNAPSHOT_HOME/config/config.toml

# add local node to the list of persistent peers
sed -i 's/persistent_peers = "'$PEERS'"/persistent_peers = "'$PEERS','$LOCAL_NODE_PEER_CONFIG'"/g' $LAYER_SNAPSHOT_HOME/config/config.toml
# open up API and RPC to outside traffic
sed -i 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' $LAYER_SNAPSHOT_HOME/config/app.toml
sed -i 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' $LAYER_SNAPSHOT_HOME/config/config.toml

# increment ports since we will have two nodes running.
    echo "adding 101 to 2nd node's port numbers to avoid port conflicts with main node"
    update_ports() {
        file="$1"
        sed -i \
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
    }

    # Update ports in LAYER_SNAPSHOT_HOME  
    echo "Updating ports..."
    update_ports $LAYER_SNAPSHOT_HOME/config/app.toml
    update_ports $LAYER_SNAPSHOT_HOME/config/client.toml
    update_ports $LAYER_SNAPSHOT_HOME/config/config.toml

# Replace auto-generated genesis file with genesis from RPC
echo "Getting genesis from local RPC....."
GENESIS_TEMP=$(sudo -u $ACTUAL_USER mktemp)
if ! curl -f "$NODE_ADDR_HTTP/genesis" | jq '.result.genesis' | sudo -u $ACTUAL_USER tee "$GENESIS_TEMP" > /dev/null; then
    echo "Error: Failed to download genesis file from $NODE_ADDR_HTTP"
    rm -f "$GENESIS_TEMP"
    exit 1
fi
sudo -u $ACTUAL_USER mv "$GENESIS_TEMP" $LAYER_SNAPSHOT_HOME/config/genesis.json


# Handle snapshot installation based on flags
echo ""
echo "================================"
echo "    INSTALLING SNAPSHOT..."
echo "================================"
echo ""
sleep 1

TEMP_DIR="$USER_HOME/tmp/layer_snapshot_extract"
VERSION_CHECK_DIR="$USER_HOME/tmp/layerd-version-check"

if [ -n "$SNAPSHOT_PATH" ]; then
    # Use pre-downloaded snapshot
    echo "Using pre-downloaded snapshot from: $SNAPSHOT_PATH"

    # Validate snapshot file exists
    if [ ! -f "$SNAPSHOT_PATH" ]; then
        echo "Error: Snapshot file not found at $SNAPSHOT_PATH"
        exit 1
    fi

    # Validate it's a tar file
    if [[ "$SNAPSHOT_PATH" != *.tar ]]; then
        echo "Warning: Snapshot file does not have .tar extension. Proceeding anyway..."
    fi

    SNAPSHOT_TAR="$SNAPSHOT_PATH"
else
    # Download the latest snapshot from https://layer-node.com
    echo "Downloading the latest snapshot from https://layer-node.com..."

    # Fetch available snapshots and parse JSON to get the latest one
    echo "Fetching available snapshots for $NETWORK..."
    SNAPSHOT_FILE=$(curl -s https://layer-node.com/files | jq -r --arg prefix "$NETWORK" '.files[] | select(.filename | contains($prefix)) | select(.filename | endswith(".tar")) | {filename: .filename, upload_time: .upload_time}' | jq -s 'sort_by(.upload_time) | reverse | .[0].filename' | tr -d '"')

    if [ -z "$SNAPSHOT_FILE" ] || [ "$SNAPSHOT_FILE" == "null" ]; then
        echo "Error: No snapshots found for $NETWORK network"
        exit 1
    fi

    echo "Latest snapshot found: $SNAPSHOT_FILE"

    # Create temporary download directory
    echo "Creating temporary directory: $TEMP_DIR"
    if ! sudo -u $ACTUAL_USER mkdir -p "$TEMP_DIR"; then
        echo "Error: Failed to create temporary directory"
        exit 1
    fi

    # Download the snapshot
    echo "Downloading snapshot (this may take a while, file size ~60-80 GB)..."
    if ! sudo -u $ACTUAL_USER curl -L -o "$TEMP_DIR/$SNAPSHOT_FILE" "https://layer-node.com/download/$SNAPSHOT_FILE"; then
        echo "Error: Failed to download snapshot"
        rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
        exit 1
    fi

    SNAPSHOT_TAR="$TEMP_DIR/$SNAPSHOT_FILE"
fi

# Create temporary extraction directory if it doesn't exist
if [ ! -d "$TEMP_DIR" ]; then
    echo "Creating temporary extraction directory: $TEMP_DIR"
    if ! sudo -u $ACTUAL_USER mkdir -p "$TEMP_DIR"; then
        echo "Error: Failed to create temporary extraction directory"
        exit 1
    fi
fi

# Extract the snapshot
echo "Extracting snapshot (this may take a while, file size ~40-80 GB)..."
cd "$TEMP_DIR"
if ! sudo -u $ACTUAL_USER tar -xf "$SNAPSHOT_TAR" --checkpoint=5000 --checkpoint-action=dot; then
    echo ""
    echo "Error: Failed to extract snapshot"
    rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
    exit 1
fi
echo ""

# Move the data files to the Layer home directory
echo "Moving blockchain data to $LAYER_SNAPSHOT_HOME/data/..."
if [ -d "$TEMP_DIR/.layer_snapshot/data" ]; then
    sudo -u $ACTUAL_USER mkdir -p "$LAYER_SNAPSHOT_HOME/data"
    sudo -u $ACTUAL_USER mv -f "$TEMP_DIR/.layer_snapshot/data/"* "$LAYER_SNAPSHOT_HOME/data/"
    echo "Blockchain data successfully installed in $LAYER_SNAPSHOT_HOME/data/"
else
    echo "Error: Expected .layer_snapshot/data directory not found in extracted snapshot"
    rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
    exit 1
fi

# Clean up temporary files
echo "Cleaning up temporary files..."
rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
echo "Snapshot installation complete!"


echo ""
echo "================================"
echo "    STARTING SNAPSHOT NODE..."
echo "================================"
echo ""
sleep 1

# start the snapshot node
echo "Starting snapshot node..."
systemctl start layer_snapshot

# Step 1: Wait for the node to begin syncing (check for "executed block app_hash")
echo "Waiting for snapshot node to begin syncing (checking for 'executed block app_hash')..."
SYNC_START_TIMEOUT=120
START_TIME=$(date +%s)
SYNC_STARTED=false

while [ $(($(date +%s) - START_TIME)) -lt $SYNC_START_TIMEOUT ]; do
    if journalctl -u layer_snapshot --since "60 seconds ago" -n 1000 | grep -q "executed block app_hash"; then
        echo "✓ Snapshot node has successfully begun syncing"
        SYNC_STARTED=true
        break
    fi
    echo "  Still waiting for sync to start... ($(($(date +%s) - START_TIME))s / ${SYNC_START_TIMEOUT}s)"
    sleep 5
done

if [ "$SYNC_STARTED" = false ]; then
    echo "Error: Snapshot node did not begin syncing within ${SYNC_START_TIMEOUT} seconds"
    echo "There may be a problem with the snapshot. Stopping layer_snapshot service..."
    systemctl stop layer_snapshot
    echo "Check the logs with: sudo journalctl -u layer_snapshot --pager-end"
    exit 1
fi

# Step 2: Wait for the node to finish syncing (check for "received complete proposal block")
echo ""
echo "Waiting for snapshot node to finish syncing (checking for 'received complete proposal block')..."
echo "This may take a while depending on how far behind the snapshot is..."


SYNC_COMPLETE=false
MAX_WAIT_SECONDS=86400
SYNC_WAIT_START=$(date +%s) # 24 hours

while [ $(($(date +%s) - SYNC_WAIT_START)) -lt $MAX_WAIT_SECONDS ]; do
    # Check if we're seeing "received complete proposal block" in recent logs
    if journalctl -u layer_snapshot --since "60 seconds ago" -n 1000 | grep -q "received complete proposal block"; then
        echo "  Detected proposal blocks being received, node appears to be syncing/synced..."
        
        # Wait 60 seconds and check again to confirm it's still receiving blocks
        echo "  Waiting 60 seconds to confirm continued block reception..."
        sleep 60
        
        # Check if we're still receiving new proposal blocks
        if journalctl -u layer_snapshot --since "60 seconds ago" -n 1000 | grep -q "received complete proposal block"; then
            echo "✓ Snapshot node has finished syncing and is receiving new blocks"
            SYNC_COMPLETE=true
            break
        else
            echo "  Node stopped receiving blocks, may still be syncing older blocks..."
        fi
    else
        ELAPSED=$(($(date +%s) - SYNC_WAIT_START))
        ELAPSED_MINS=$((ELAPSED / 60))
        echo "  Still syncing... (${ELAPSED_MINS} minutes elapsed)"
        sleep 30
    fi
done

if [ "$SYNC_COMPLETE" = false ]; then
    echo "Error: Snapshot node did not finish syncing within ${MAX_WAIT_SECONDS} seconds"
    echo "Check the logs with: sudo journalctl -fu layer_snapshot"
        systemctl stop layer_snapshot
        exit 1
fi

echo ""
echo "Snapshot node sync complete! Ready to replace main node data..."


echo "Send discord alert if webhook is configured..."
if [ -n "$DISCORD_WEBHOOK" ]; then
    echo "Sending discord alert..."
    discord_alert "2nd Node is synced and ready to replace main node data!"
else
    echo "No Discord webhook configured, skipping alert"
fi
echo "--------------------------------"

echo "press enter begin moving data from $LAYER_SNAPSHOT_HOME/data to $LAYER_HOME/data..."
echo "The old data will be moved to ~/tmp/layer_data/"
echo "and the new data will be moved to $LAYER_HOME/data/"
echo "press enter to continue..."
read -p ""

echo ""
echo "===================================================="
echo "STOPPING SERVICES BEFORE HANDLING CHAIN DATA..."
echo "===================================================="
echo ""
sleep 1

echo "Stopping layer service..."
systemctl stop layer
sleep 5
echo "checking if layer service is stopped..."
if journalctl -u layer -n 10 --no-pager | grep -q "Stopped layer.service"; then
    echo "✓ layer service is stopped"
else
    echo "Error: layer service is not stopped. "
    echo "Please check the logs with: sudo journalctl -u layer --pager-end"
    echo "To stop the service manually, in a separate terminal run: sudo systemctl stop layer"
    echo "press enter to continue once they are stopped..."
    read -p ""
    exit 1
fi
echo "--------------------------------"

echo "Stopping layer_snapshot service..."
systemctl stop layer_snapshot
sleep 5
echo "checking if snapshot service is stopped..."
if journalctl -u layer_snapshot -n 10 --no-pager | grep -q "Stopped layer_snapshot.service"; then
    echo "✓ layer_snapshot service is stopped"
else
    echo "Error: layer_snapshot service is not stopped..."
    echo "Please check the logs with: sudo journalctl -u layer_snapshot --pager-end"
    echo "To stop the service manually, in a separate terminal run: sudo systemctl stop layer_snapshot"
    echo "press enter to continue once they are stopped..."
    read -p ""
    exit 1
fi
echo "--------------------------------"


echo ""
echo "========================================="
echo "  REPLACING MAIN NODE DATA..."
echo "========================================="
echo ""
sleep 1

# Create tmp directory if it doesn't exist
echo "Ensuring tmp directory exists..."
echo "DEBUG: Checking $USER_HOME/tmp before mkdir..."
if [ -e "$USER_HOME/tmp" ]; then
    echo "DEBUG: $USER_HOME/tmp exists"
    ls -ld "$USER_HOME/tmp"
    if [ -d "$USER_HOME/tmp" ]; then
        echo "DEBUG: It's a directory"
    else
        echo "DEBUG: WARNING - It's NOT a directory (file or symlink?)"
        file "$USER_HOME/tmp"
    fi
else
    echo "DEBUG: $USER_HOME/tmp does not exist yet"
fi

echo "DEBUG: Running mkdir -p $USER_HOME/tmp/layer_data"
if sudo -u "$ACTUAL_USER" mkdir -p "$USER_HOME/tmp/layer_data"; then
    echo "DEBUG: mkdir succeeded"
else
    echo "DEBUG: mkdir FAILED with exit code $?"
    exit 1
fi

echo "DEBUG: Verifying directory was created..."
if [ -d "$USER_HOME/tmp/layer_data" ]; then
    echo "DEBUG: ✓ $USER_HOME/tmp/layer_data exists and is a directory"
    ls -ld "$USER_HOME/tmp/layer_data"
else
    echo "DEBUG: ✗ $USER_HOME/tmp/layer_data was NOT created!"
    echo "DEBUG: Checking what exists at the path..."
    ls -la "$USER_HOME/tmp/" 2>/dev/null || echo "DEBUG: Cannot list $USER_HOME/tmp/"
    exit 1
fi

# move the big old chain data to ~/tmp/layer_data
echo "moving $LAYER_HOME/data to ~/tmp/layer_data..."
echo "DEBUG: Source directory contents ($LAYER_HOME/data/):"
ls -la "$LAYER_HOME/data/" 2>&1 || echo "DEBUG: Cannot list source directory"
echo ""
echo "DEBUG: Target directory contents before move ($USER_HOME/tmp/layer_data/):"
ls -la "$USER_HOME/tmp/layer_data/" 2>&1 || echo "DEBUG: Cannot list target directory"
echo ""

if ls "$LAYER_HOME/data/"* >/dev/null 2>&1; then
    echo "DEBUG: Files found in source, attempting move..."
    echo "DEBUG: Running: sudo -u $ACTUAL_USER mv -f $LAYER_HOME/data/* $USER_HOME/tmp/layer_data/"
    if sudo -u "$ACTUAL_USER" mv -f "$LAYER_HOME/data/"* "$USER_HOME/tmp/layer_data/"; then
        echo "DEBUG: mv command succeeded"
    else
        echo "DEBUG: mv command FAILED with exit code $?"
        exit 1
    fi
else
    echo "Warning: No files found in $LAYER_HOME/data/ to move"
fi
echo "checking if the data was moved..."
echo "DEBUG: Target directory contents after move:"
ls -la $USER_HOME/tmp/layer_data
echo "--------------------------------"
echo ""

echo "============================================================"
echo "CRITICAL OPERATION: Moving snapshot data to live chain data"
echo "============================================================"
echo ""
echo "moving data from $LAYER_SNAPSHOT_HOME/data to $LAYER_HOME/data..."
echo ""
echo "DEBUG: Source directory contents ($LAYER_SNAPSHOT_HOME/data/):"
ls -la "$LAYER_SNAPSHOT_HOME/data/" 2>&1 || echo "DEBUG: Cannot list source directory"
echo ""
echo "DEBUG: Target directory contents BEFORE move ($LAYER_HOME/data/):"
ls -la "$LAYER_HOME/data/" 2>&1 || echo "DEBUG: Cannot list target directory (should be empty after backup)"
echo ""

if ls "$LAYER_SNAPSHOT_HOME/data/"* >/dev/null 2>&1; then
    echo "DEBUG: Files found in snapshot source, attempting move..."
    echo "DEBUG: Running: sudo -u $ACTUAL_USER mv -f $LAYER_SNAPSHOT_HOME/data/* $LAYER_HOME/data/"
    if sudo -u "$ACTUAL_USER" mv -f "$LAYER_SNAPSHOT_HOME/data/"* "$LAYER_HOME/data/"; then
        echo "DEBUG: mv command succeeded"
    else
        echo "DEBUG: mv command FAILED with exit code $?"
        echo "DEBUG: Checking if target directory exists..."
        ls -ld "$LAYER_HOME/data/" 2>&1 || echo "DEBUG: Target directory doesn't exist!"
        echo "DEBUG: Checking target directory permissions..."
        stat "$LAYER_HOME/data/" 2>&1 || echo "DEBUG: Cannot stat target directory"
        exit 1
    fi
else
    echo "Error: No files found in $LAYER_SNAPSHOT_HOME/data/ to move"
    echo "DEBUG: This is critical - the snapshot data should exist here!"
    echo "DEBUG: Checking if directory exists at all..."
    ls -ld "$LAYER_SNAPSHOT_HOME/data/" 2>&1 || echo "DEBUG: Directory doesn't exist"
    echo "DEBUG: Checking parent directory..."
    ls -la "$LAYER_SNAPSHOT_HOME/" 2>&1 || echo "DEBUG: Cannot list $LAYER_SNAPSHOT_HOME/"
    exit 1
fi

echo ""
echo "DEBUG: Target directory contents AFTER move ($LAYER_HOME/data/):"
ls -la "$LAYER_HOME/data/" 2>&1 || echo "DEBUG: Cannot list target directory after move!"
echo ""
echo "DEBUG: Source directory contents after move (should be empty):"
ls -la "$LAYER_SNAPSHOT_HOME/data/" 2>&1 || echo "DEBUG: Cannot list source directory"
echo "--------------------------------"
echo ""

# cp priv_validator_state.json to $LAYER_HOME/data
echo "============================================================"
echo "CRITICAL OPERATION: Restoring priv_validator_state.json"
echo "============================================================"
echo ""
echo "copying priv_validator_state.json to $LAYER_HOME/data..."
echo ""
echo "DEBUG: Checking source file exists ($USER_HOME/tmp/layer_data/priv_validator_state.json):"
if [ -f "$USER_HOME/tmp/layer_data/priv_validator_state.json" ]; then
    echo "DEBUG: ✓ Source file exists"
    ls -la "$USER_HOME/tmp/layer_data/priv_validator_state.json"
    echo "DEBUG: File contents:"
    cat "$USER_HOME/tmp/layer_data/priv_validator_state.json"
    echo ""
else
    echo "DEBUG: ✗ Source file NOT FOUND!"
    echo "DEBUG: Listing $USER_HOME/tmp/layer_data/:"
    ls -la "$USER_HOME/tmp/layer_data/" 2>&1 || echo "DEBUG: Cannot list directory"
    echo "ERROR: priv_validator_state.json not found in backup! This is critical!"
    exit 1
fi

echo ""
echo "DEBUG: Running: sudo -u $ACTUAL_USER cp -f $USER_HOME/tmp/layer_data/priv_validator_state.json $LAYER_HOME/data/priv_validator_state.json"
if sudo -u "$ACTUAL_USER" cp -f "$USER_HOME/tmp/layer_data/priv_validator_state.json" "$LAYER_HOME/data/priv_validator_state.json"; then
    echo "DEBUG: cp command succeeded"
else
    echo "DEBUG: cp command FAILED with exit code $?"
    exit 1
fi

echo ""
echo "DEBUG: Verifying copied file:"
if [ -f "$LAYER_HOME/data/priv_validator_state.json" ]; then
    echo "DEBUG: ✓ File exists at destination"
    ls -la "$LAYER_HOME/data/priv_validator_state.json"
    echo "DEBUG: File contents:"
    cat "$LAYER_HOME/data/priv_validator_state.json"
    echo ""
else
    echo "DEBUG: ✗ File NOT FOUND at destination after copy!"
    exit 1
fi
echo "--------------------------------"

echo "New chain data imported!"

echo ""
echo "=================================================="
echo "STARTING MAIN NODE WITH IMPORTED CHAIN DATA..."
echo "=================================================="
echo ""
sleep 1

echo "starting layer service..."
systemctl start layer

# check if the layer service starts syncing successfully
echo "checking if the layer service starts syncing successfully..."
sleep 60
if journalctl -u layer --since "60 seconds ago" -n 1000 | grep -q "executed block app_hash"; then
    echo "✓ Layer service has successfully begun syncing"
else
    echo "Error: Layer service did not begin syncing within 10 seconds"
    exit 1
fi
echo "--------------------------------"

# check if the layer service catches up to the latest block
echo "checking if the layer service catches up to the latest block..."
sleep 10
if journalctl -u layer --since "60 seconds ago" -n 1000 | grep -q "received complete proposal block"; then
    echo "✓ Layer service has successfully caught up to the latest block"
else
    echo "Error: Layer service did not catch up to the latest block within 10 seconds"
    echo "Please check the logs with: sudo journalctl -u layer --pager-end"
    exit 1
fi

# check layerd status to see if catching_up is false
echo "checking if the layer service is not catching up..."
sleep 10
MAX_RETRIES=5
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    CATCHING_UP=$("$LAYERD_PATH" status --home "$LAYER_HOME" --node "$NODE_ADDR_TCP" 2>/dev/null | jq -r '.sync_info.catching_up' 2>/dev/null || echo "error")
    if [ "$CATCHING_UP" == "false" ]; then
        echo "✓ Layer service is fully synced"
        break
    elif [ "$CATCHING_UP" == "error" ]; then
        echo "Warning: Failed to get sync status, retrying..."
    fi
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            echo "Layer service is still catching up (attempt $RETRY_COUNT/$MAX_RETRIES). Checking again in 60 seconds..."
            sleep 60
        else
            echo "Error: Layer service is still catching up after $MAX_RETRIES attempts"
            exit 1
    fi
done

echo "Layer service has successfully caught up to the latest block!"

# Send final success Discord alert
if [ -n "$DISCORD_WEBHOOK" ]; then
    discord_alert "✅ Layer service has successfully caught up to the latest block! Resync complete."
fi

echo "Have a great day!"

