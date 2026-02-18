#!/bin/bash

# Check if script is run with sudo
if [ "$EUID" -eq 0 ] || [ -n "$SUDO_USER" ]; then
    echo "Error: This script should NOT be run with sudo or as root."
    echo "Please run as a regular user: ./install_layer.sh <NETWORK> [OPTIONS]"
    exit 1
fi

# Stop execution if any command fails
set -e

# Function to check required dependencies
check_dependencies() {
    local deps=("jq" "curl" "wget" "tar")
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done
    
    if [ ${#missing[@]} -gt 0 ]; then
        echo "Error: Required dependencies are not installed: ${missing[*]}"
        echo "Please install them and try again."
        echo ""
        if [ "$OS_TYPE" == "Linux" ]; then
            echo "On Ubuntu/Debian: sudo apt-get install ${missing[*]}"
            echo "On Fedora/RHEL: sudo dnf install ${missing[*]}"
        elif [ "$OS_TYPE" == "Darwin" ]; then
            echo "On macOS: brew install ${missing[*]}"
        fi
        exit 1
    fi
}

# Function to check available disk space
check_disk_space() {
    local required_gb=128
    
    if [ "$OS_TYPE" == "Darwin" ]; then
        # macOS uses different df output format
        local available=$(df -g "$HOME" | awk 'NR==2 {print $4}')
    else
        # Linux
        local available=$(df -BG "$HOME" | awk 'NR==2 {print $4}' | sed 's/G//')
    fi
    
    if [ "$available" -lt "$required_gb" ]; then
        echo "Warning: Low disk space detected."
        echo "Available: ${available}GB, Recommended: ${required_gb}GB"
        read -p "Do you want to continue anyway? (y/n): " disk_choice
        case "$disk_choice" in
            y|Y|yes|Yes|YES)
                echo "Continuing with limited disk space..."
                ;;
            *)
                echo "Exiting. Please free up disk space and try again."
                exit 1
                ;;
        esac
    fi
}

# Cross-platform sed in-place function
sed_inplace() {
    if [ "$OS" == "mac" ]; then
        sed -i '' "$@"
    else
        sed -i "$@"
    fi
}

# Get username with fallback
get_username() {
    if command -v logname &> /dev/null && logname &> /dev/null; then
        logname
    else
        whoami
    fi
}

# Detect operating system
OS_TYPE=$(uname -s)
case "$OS_TYPE" in
    Linux*)
        OS="linux"
        SHELL_RC="$HOME/.bashrc"
        USER_HOME="/home/$(get_username)"
        ;;
    Darwin*)
        OS="mac"
        SHELL_RC="$HOME/.zshrc"
        USER_HOME="/Users/$(get_username)"
        ;;
    *)
        echo "Error: Unsupported operating system: $OS_TYPE"
        echo "This script only supports Linux and macOS"
        exit 1
        ;;
esac

echo "Detected operating system: $OS"

# Check dependencies early
check_dependencies

# Initialize variables
SNAPSHOT_PATH=""
SKIP_SNAPSHOT=false
TEMP_DIR=""
VERSION_CHECK_DIR=""
SNAPSHOT_DOWNLOAD_DIR=""
DOWNLOADED_SNAPSHOT_FILE=""

# Cleanup function for temporary directories (preserves downloaded snapshots)
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        echo "Cleaning up temporary extraction directory: $TEMP_DIR"
        rm -rf "$TEMP_DIR" 2>/dev/null || true
    fi
    if [ -n "$VERSION_CHECK_DIR" ] && [ -d "$VERSION_CHECK_DIR" ]; then
        rm -rf "$VERSION_CHECK_DIR" 2>/dev/null || true
    fi
    # Note: Downloaded snapshot files are intentionally preserved in ~/layer/snapshots/
    # to avoid re-downloading on failure. User will be prompted to delete on success.
    if [ -n "$DOWNLOADED_SNAPSHOT_FILE" ] && [ -f "$DOWNLOADED_SNAPSHOT_FILE" ]; then
        echo ""
        echo "Note: Downloaded snapshot preserved at: $DOWNLOADED_SNAPSHOT_FILE"
        echo "You can reuse it with: ./install_layer.sh $NETWORK --snapshot \"$DOWNLOADED_SNAPSHOT_FILE\""
    fi
}

# Set trap to cleanup on exit, interrupt, or termination
trap cleanup EXIT INT TERM

# Check if NETWORK argument is provided
if [ $# -eq 0 ]; then
    echo "Error: No arguments provided!"
    echo "Usage: $0 <NETWORK> [NODE_MONIKER] [OPTIONS]"
    echo "  NETWORK: 'mainnet' or 'palmito' (required)"
    echo "  NODE_MONIKER: Nickname for your node and account (optional)"
    echo ""
    echo "Options:"
    echo "  --snapshot <path>   Use a pre-downloaded snapshot file"
    echo "  --no-snapshot       Skip snapshot installation entirely"
    exit 1
fi

# Get the NETWORK argument
NETWORK="$1"

# Validate NETWORK argument
if [ "$NETWORK" != "mainnet" ] && [ "$NETWORK" != "palmito" ]; then
    echo "Error: Invalid NETWORK value '$NETWORK'"
    echo "NETWORK must be either 'mainnet' or 'palmito'"
    echo "Usage: $0 <NETWORK> [NODE_MONIKER] [OPTIONS]"
    exit 1
fi

echo "Running setup for network: $NETWORK"

# Parse remaining arguments
shift  # Remove NETWORK from arguments
NODE_MONIKER=""

# Check if second argument is a flag or moniker
if [ $# -gt 0 ] && [[ "$1" != --* ]]; then
    NODE_MONIKER="$1"
    shift
fi

# Parse optional flags
while [ $# -gt 0 ]; do
    case "$1" in
        --snapshot)
            if [ -z "$2" ] || [[ "$2" == --* ]]; then
                echo "Error: --snapshot flag requires a path argument"
                exit 1
            fi
            SNAPSHOT_PATH="$2"
            shift 2
            ;;
        --no-snapshot)
            SKIP_SNAPSHOT=true
            shift
            ;;
        *)
            echo "Error: Unknown option '$1'"
            echo "Usage: $0 <NETWORK> [NODE_MONIKER] [OPTIONS]"
            echo "Options:"
            echo "  --snapshot <path>   Use a pre-downloaded snapshot file"
            echo "  --no-snapshot       Skip snapshot installation entirely"
            exit 1
            ;;
    esac
done

# Validate snapshot flags aren't used together
if [ -n "$SNAPSHOT_PATH" ] && [ "$SKIP_SNAPSHOT" = true ]; then
    echo "Error: Cannot use both --snapshot and --no-snapshot flags"
    exit 1
fi

# Check if NODE_MONIKER was provided
if [ -z "$NODE_MONIKER" ]; then
    echo ""
    echo "No account name provided. Skipping account setup. (this is fine)"
    echo "You can create an account later using the ./layerd keys add command."
else
    echo "Tellor account with name: $NODE_MONIKER will be created..."
fi

# init variables for mainnet and palmito
LAYERD_TAG_MAINNET="v6.1.1"
LAYERD_TAG_PALMITO="v6.1.2"
MAINNET_LAYER_NODE_URL=https://mainnet.tellorlayer.com/rpc/
PALMITO_LAYER_NODE_URL=https://node-palmito.tellorlayer.com/rpc/
MAINNET_RPC_NODE_ID=cbb94e01df344fdfdee1fdf2f9bb481712e7ef8d
PALMITO_RPC_NODE_ID=ac7c10dc3de67c4394271c564671eeed4ac6f0e0
MAINNET_KEYRING_BACKEND="test"
PALMITO_KEYRING_BACKEND="test"
MAINNET_PEERS="5a9db46eceb055c9238833aa54e15a2a32a09c9a@54.67.36.145:26656,f2644778a8a2ca3b55ec65f1b7799d32d4a7098e@54.149.160.93:26656,2904aa32501548e127d3198c8f5181fb4d67bbe6@18.116.23.104:26656,7fd4d34f3b19c41218027d3b91c90d073ab2ba66@54.221.149.61:26656,2b8af463a1f0e84aec6e4dbf3126edf3225df85e@13.52.231.70:26656,cbb94e01df344fdfdee1fdf2f9bb481712e7ef8d@54.234.103.186:26656"
PALMITO_PEERS="ac7c10dc3de67c4394271c564671eeed4ac6f0e0@34.229.148.107:26656,8d19cdf430e491d6d6106863c4c466b75a17088a@54.153.125.203:26656,c7b175a5bafb35176cdcba3027e764a0dbd0811c@34.219.95.82:26656,05105e8bb28e8c5ace1cecacefb8d4efb0338ec6@18.218.114.74:26656,705f6154c6c6aeb0ba36c8b53639a5daa1b186f6@3.80.39.230:26656"
MAINNET_LAYER_HOME="$USER_HOME/.layer"
PALMITO_LAYER_HOME="$USER_HOME/.layer_palmito"

if [ "$NETWORK" == "mainnet" ]; then
    LAYERD_TAG=$LAYERD_TAG_MAINNET
    LAYER_NODE_URL=$MAINNET_LAYER_NODE_URL
    RPC_NODE_ID=$MAINNET_RPC_NODE_ID
    KEYRING_BACKEND=$MAINNET_KEYRING_BACKEND
    PEERS=$MAINNET_PEERS
    LAYER_HOME=$MAINNET_LAYER_HOME
    CHAIN_ID="tellor-1"
elif [ "$NETWORK" == "palmito" ]; then
    LAYERD_TAG=$LAYERD_TAG_PALMITO
    LAYER_NODE_URL=$PALMITO_LAYER_NODE_URL
    RPC_NODE_ID=$PALMITO_RPC_NODE_ID
    KEYRING_BACKEND=$PALMITO_KEYRING_BACKEND
    PEERS=$PALMITO_PEERS
    LAYER_HOME=$PALMITO_LAYER_HOME
    CHAIN_ID="layertest-4"
fi

# set cosmovisor environment variables for init command (dynamic based on network)
export DAEMON_NAME=layerd
export DAEMON_HOME=$LAYER_HOME
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_POLL_INTERVAL=300ms
export UNSAFE_SKIP_BACKUP=true
export DAEMON_PREUPGRADE_MAX_RETRIES=0

# check if layer home directory exists
if [ -d "$LAYER_HOME" ]; then
    echo "--------------------------------"
    echo ""
    echo "Error: $LAYER_HOME directory already exists. This script is for new setups only."
    echo "If you want to reconfigure, please backup and remove the existing $LAYER_HOME directory first."
    echo "--------------------------------"
    exit 1
fi

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
echo "Node Quick-Installer for Linux"
echo "--------------------------------"
echo "This is a quick-installer for Tellor Layer."
echo "This script will: "
echo "  1) download the latest layerd and cosmovisor binaries."
echo "  2) initialize the layer node. (home dir: ~/.layer)."
if [ "$OS" == "linux" ]; then
    echo "  3) Add cosmovisor environment variables to .bashrc."
else
    echo "  3) Add cosmovisor environment variables to .zshrc."
fi
if [ -n "$SNAPSHOT_PATH" ]; then
    echo "  4) Install pre-downloaded snapshot from: $SNAPSHOT_PATH"
elif [ "$SKIP_SNAPSHOT" = true ]; then
    echo "  4) Skip snapshot installation (you will need to configure sync yourself.)"
else
    echo "  4) Download and install the latest pre-built snapshot from https://layer-node.com."
fi
echo ""
echo "--------------------------------"
echo ""
echo "Network: $NETWORK"
echo "Layer Node URL: $LAYER_NODE_URL"
echo "Keyring Backend: $KEYRING_BACKEND"
echo "Layer Home: $LAYER_HOME"
echo "layerd binary version: $LAYERD_TAG"
echo ""
echo "--------------------------------"
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

# Check disk space before proceeding with downloads
check_disk_space

# download the current layerd binary
echo "Checking for layerd binary for $NETWORK..."
mkdir -p ~/layer/binaries && cd ~/layer/binaries

echo ""
echo "================================"
echo "  GATHERING BINARIES..."
echo "================================"
echo ""
sleep 1

# Determine the binary file name based on OS
if [ "$OS_TYPE" == "Darwin" ]; then
    BINARY_FILE="layer_Darwin_arm64.tar.gz"
else
    BINARY_FILE="layer_Linux_x86_64.tar.gz"
fi

# Check if binary already exists and verify version
if [ -d "$LAYERD_TAG" ] && [ -f "$LAYERD_TAG/layerd" ]; then
    echo "Binary directory $LAYERD_TAG already exists. Checking version..."
    EXISTING_VERSION=$(cd $LAYERD_TAG && $USER_HOME/layer/binaries/$LAYERD_TAG/layerd version --home $USER_HOME/tmp/layerd-version-check 2>&1 | tr -d '\n')
    # Normalize versions by removing 'v' prefix for comparison
    NORMALIZED_EXISTING="${EXISTING_VERSION#v}"
    NORMALIZED_REQUIRED="${LAYERD_TAG#v}"
    echo "Comparing versions: existing=$NORMALIZED_EXISTING, required=$NORMALIZED_REQUIRED"
    if [ "$NORMALIZED_EXISTING" == "$NORMALIZED_REQUIRED" ]; then
        echo "Existing binary version matches required version ($LAYERD_TAG). Skipping download."
        rm -rf $USER_HOME/.layer
    else
        echo "Existing binary version ($EXISTING_VERSION) does not match required version ($LAYERD_TAG)."
        echo "Downloading correct version..."
        rm -rf $LAYERD_TAG
        mkdir $LAYERD_TAG && cd $LAYERD_TAG
        if ! wget https://github.com/tellor-io/layer/releases/download/$LAYERD_TAG/$BINARY_FILE; then
            echo "Error: Failed to download layerd binary"
            exit 1
        fi
        if ! tar -xvzf $BINARY_FILE; then
            echo "Error: Failed to extract layerd binary"
            exit 1
        fi
        rm $BINARY_FILE
        rm -rf $USER_HOME/.layer
    fi
else
    echo "Binary not found. Downloading layerd binary for $NETWORK..."
    mkdir -p $LAYERD_TAG && cd $LAYERD_TAG
    if ! wget https://github.com/tellor-io/layer/releases/download/$LAYERD_TAG/$BINARY_FILE; then
        echo "Error: Failed to download layerd binary"
        exit 1
    fi
    if ! tar -xvzf $BINARY_FILE; then
        echo "Error: Failed to extract layerd binary"
        exit 1
    fi
    rm $BINARY_FILE
fi

# download the current cosmovisor binary (skip on macOS)
# https://github.com/cosmos/cosmos-sdk/releases/tag/cosmovisor%2Fv1.3.0
if [ "$OS_TYPE" == "Darwin" ]; then
    echo "Skipping cosmovisor binary download on macOS..."
else
    if [ -f ~/layer/binaries/cosmovisor/cosmovisor ]; then
        echo "Cosmovisor binary already exists. Skipping download."
    else
        echo "Downloading cosmovisor binary..."
        mkdir -p ~/layer/binaries/cosmovisor && cd ~/layer/binaries/cosmovisor
        if ! wget https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2Fv1.3.0/cosmovisor-v1.3.0-linux-amd64.tar.gz; then
            echo "Error: Failed to download cosmovisor binary"
            exit 1
        fi
        if ! tar -xvzf cosmovisor-v1.3.0-linux-amd64.tar.gz; then
            echo "Error: Failed to extract cosmovisor binary"
            exit 1
        fi
        rm cosmovisor-v1.3.0-linux-amd64.tar.gz
    fi
fi

LAYERD_PATH="$USER_HOME/layer/binaries/$LAYERD_TAG/layerd"

# initialize layer directory
echo "Initializing layer directory..."

# Set default NODE_MONIKER if not provided
if [ -z "$NODE_MONIKER" ]; then
    echo "Running: $LAYERD_PATH init layer --chain-id $CHAIN_ID --home $LAYER_HOME"
    $LAYERD_PATH init layer --chain-id $CHAIN_ID --home $LAYER_HOME
    echo "No node moniker provided, using default 'layer'..."
else
    echo "Running: $LAYERD_PATH init $NODE_MONIKER --chain-id $CHAIN_ID --home $LAYER_HOME"
    $LAYERD_PATH init $NODE_MONIKER --chain-id $CHAIN_ID --home $LAYER_HOME
fi

# check if RPC node is running and verify node id
export LAYER_NODE_ID=$($LAYERD_PATH status --node $LAYER_NODE_URL | jq -r '.node_info.id')
if [ "$LAYER_NODE_ID" != "$RPC_NODE_ID" ]; then
    echo "Error: RPC node is not running or the node id does not match the expected node id."
    echo "Expected node id: $RPC_NODE_ID"
    echo "Actual node id: $LAYER_NODE_ID"
    exit 1
fi

echo ""
echo "================================"
echo "  INITIALIZING NODE CONFIGS..."
echo "================================"
echo ""
sleep 1

# change denom, chain id, and timeout commit in config files
echo "Changing configs for $NETWORK..."
sed_inplace 's/[0-9]\+stake/0loya/g' $LAYER_HOME/config/app.toml
sed_inplace 's/^chain-id = .*$/chain-id = "'$CHAIN_ID'"/g' $LAYER_HOME/config/client.toml
sed_inplace 's/timeout_commit = "5s"/timeout_commit = "1s"/' $LAYER_HOME/config/config.toml
sed_inplace 's/^cors_allowed_origins = \[\]/cors_allowed_origins = \["\*"\]/g' $LAYER_HOME/config/config.toml
sed_inplace 's/^keyring-backend = "os"/keyring-backend = "'$KEYRING_BACKEND'"/g' $LAYER_HOME/config/client.toml
sed_inplace 's/persistent_peers = ""/persistent_peers = "'$PEERS'"/g' $LAYER_HOME/config/config.toml

# change snapshot configs to match network normal
# This makes statesync work better across the whole network
sed_inplace 's/^snapshot-interval = 0/snapshot-interval = 32000/g' $LAYER_HOME/config/app.toml
sed_inplace 's/^snapshot-keep-recent = 2/snapshot-keep-recent = 5/g' $LAYER_HOME/config/app.toml

# open up API and RPC to outside traffic
sed_inplace 's/^address = "tcp:\/\/localhost:1317"/address = "tcp:\/\/0.0.0.0:1317"/g' $LAYER_HOME/config/app.toml
sed_inplace 's/^laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/g' $LAYER_HOME/config/config.toml

# Replace auto-generated genesis file with genesis from RPC
rm -f $LAYER_HOME/config/genesis.json
echo "Getting genesis from RPC....."
if ! curl -f "$LAYER_NODE_URL/genesis" | jq '.result.genesis' > $LAYER_HOME/config/genesis.json; then
    echo "Error: Failed to download genesis file from $LAYER_NODE_URL"
    exit 1
fi

# Check if user wants to create or import an account
if [ -z "$NODE_MONIKER" ]; then
    echo "Skipping account setup..."
else
    echo ""
    echo "================================"
    echo "    CREATING TELLOR ACCOUNT..."
    echo "================================"
    echo ""
    sleep 1
    
    # Ask user if they want to import an existing mnemonic
    while true; do
        read -p "Do you have an existing mnemonic you would like to import? (y/n): " import_choice
        
        case "$import_choice" in
            y|Y|yes|Yes|YES)
                echo ""
                $LAYERD_PATH keys add $NODE_MONIKER --recover --keyring-backend $KEYRING_BACKEND
                echo "--------------------------------"
                echo "Account successfully imported!"
                echo "--------------------------------"
                break
                ;;
            n|N|no|No|NO)
                echo ""
                echo "Creating new Tellor account for $NODE_MONIKER..."
                $LAYERD_PATH keys add $NODE_MONIKER --keyring-backend $KEYRING_BACKEND
                echo "--------------------------------"
                echo "Please save your mnemonic in a secure location!"
                read -p "Press Enter when you have it written down..."
                echo "--------------------------------"
                break
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
fi

# clear the terminal
clear

# Function to extract and install snapshot
extract_and_install_snapshot() {
    local snapshot_file="$1"
    local temp_dir="$2"
    local is_downloaded="$3"  # "true" if this was downloaded (not user-provided)
    
    # Extract the snapshot
    echo "Extracting snapshot (this may take a while, file size ~40-80 GB)..."
    cd "$temp_dir"
    if ! tar -xf "$snapshot_file" --checkpoint=5000 --checkpoint-action=dot; then
        echo ""
        echo "Error: Failed to extract snapshot"
        echo ""
        if [ "$is_downloaded" = "true" ]; then
            echo "The downloaded snapshot has been preserved at:"
            echo "  $snapshot_file"
            echo ""
            echo "You can retry installation with:"
            echo "  ./install_layer.sh $NETWORK --snapshot \"$snapshot_file\""
        fi
        # Clean up extraction directory but preserve the snapshot file
        rm -rf "$temp_dir" "$VERSION_CHECK_DIR"
        exit 1
    fi
    echo ""
    
    # Move the data files to the Layer home directory
    echo "Moving blockchain data to $LAYER_HOME/data/..."
    if [ -d "$temp_dir/.layer_snapshot/data" ]; then
        # Preserve priv_validator_state.json before replacing data directory
        if [ -f "$LAYER_HOME/data/priv_validator_state.json" ]; then
            cp "$LAYER_HOME/data/priv_validator_state.json" "$temp_dir/priv_validator_state.json.bak"
        fi
        # Remove the initialized data directory and replace with snapshot data
        rm -rf "$LAYER_HOME/data"
        mv "$temp_dir/.layer_snapshot/data" "$LAYER_HOME/data"
        # Restore priv_validator_state.json
        if [ -f "$temp_dir/priv_validator_state.json.bak" ]; then
            mv "$temp_dir/priv_validator_state.json.bak" "$LAYER_HOME/data/priv_validator_state.json"
        fi
        echo "Blockchain data successfully installed"
    else
        echo "Error: Expected .layer_snapshot/data directory not found in extracted snapshot"
        echo ""
        if [ "$is_downloaded" = "true" ]; then
            echo "The downloaded snapshot has been preserved at:"
            echo "  $snapshot_file"
            echo ""
            echo "You can retry installation with:"
            echo "  ./install_layer.sh $NETWORK --snapshot \"$snapshot_file\""
        fi
        # Clean up extraction directory but preserve the snapshot file
        rm -rf "$temp_dir" "$VERSION_CHECK_DIR"
        exit 1
    fi
    
    # Clean up temporary extraction directory
    echo "Cleaning up temporary extraction directory..."
    rm -rf "$temp_dir" "$VERSION_CHECK_DIR"
    
    # If this was a downloaded snapshot, ask user if they want to delete it
    if [ "$is_downloaded" = "true" ]; then
        echo ""
        echo "Snapshot installation complete!"
        echo ""
        echo "The downloaded snapshot file is stored at:"
        echo "  $snapshot_file"
        echo ""
        while true; do
            read -p "Do you want to delete the snapshot file to free up disk space? (y/n): " delete_choice
            case "$delete_choice" in
                y|Y|yes|Yes|YES)
                    echo "Deleting snapshot file..."
                    rm -f "$snapshot_file"
                    # Remove the snapshots directory if empty
                    rmdir "$SNAPSHOT_DOWNLOAD_DIR" 2>/dev/null || true
                    echo "Snapshot file deleted."
                    # Clear the variable so cleanup doesn't print the preservation message
                    DOWNLOADED_SNAPSHOT_FILE=""
                    break
                    ;;
                n|N|no|No|NO)
                    echo "Keeping snapshot file. You can delete it later or reuse it with:"
                    echo "  ./install_layer.sh $NETWORK --snapshot \"$snapshot_file\""
                    # Clear the variable so cleanup doesn't print the preservation message
                    DOWNLOADED_SNAPSHOT_FILE=""
                    break
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
    else
        echo "Snapshot installation complete!"
    fi
}

# Handle snapshot installation based on flags
if [ "$SKIP_SNAPSHOT" = true ]; then
    echo "Skipping snapshot installation (--no-snapshot flag provided)"
    echo "Note: Your node will need to sync from genesis or use state sync"
elif [ -n "$SNAPSHOT_PATH" ]; then
    echo ""
    echo "================================"
    echo "    INSTALLING SNAPSHOT..."
    echo "================================"
    echo ""
    sleep 1
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

    # Create temporary extraction directory
    TEMP_DIR="$USER_HOME/tmp/layer_snapshot_extract"
    VERSION_CHECK_DIR="$USER_HOME/tmp/layerd-version-check"
    echo "Creating temporary extraction directory: $TEMP_DIR"
    if ! mkdir -p "$TEMP_DIR"; then
        echo "Error: Failed to create temporary extraction directory"
        exit 1
    fi

    # Extract and install the snapshot (user-provided, not downloaded)
    extract_and_install_snapshot "$SNAPSHOT_PATH" "$TEMP_DIR" "false"
else
    # Default behavior: Download and install the latest pre-built snapshot from https://layer-node.com
    echo ""
    echo "================================"
    echo "    INSTALLING SNAPSHOT..."
    echo "================================"
    echo ""
    sleep 1
    echo "Downloading and installing the latest pre-built snapshot from https://layer-node.com..."
    
    # Determine the network prefix for filtering snapshots
    if [ "$NETWORK" == "mainnet" ]; then
        SNAPSHOT_PREFIX="tellor"
    elif [ "$NETWORK" == "palmito" ]; then
        SNAPSHOT_PREFIX="layertest"
    fi
    
    # Fetch available snapshots and parse JSON to get the latest one
    echo "Fetching available snapshots for $NETWORK..."
    SNAPSHOT_FILE=$(curl -s https://layer-node.com/files | jq -r --arg prefix "$SNAPSHOT_PREFIX" '.files[] | select(.filename | contains($prefix)) | select(.filename | endswith(".tar")) | {filename: .filename, upload_time: .upload_time}' | jq -s 'sort_by(.upload_time) | reverse | .[0].filename' | tr -d '"')
    
    if [ -z "$SNAPSHOT_FILE" ] || [ "$SNAPSHOT_FILE" == "null" ]; then
        echo "Error: No snapshots found for $NETWORK network"
        exit 1
    fi
    
    echo "Latest snapshot found: $SNAPSHOT_FILE"
    
    # Create snapshot download directory (persistent location)
    SNAPSHOT_DOWNLOAD_DIR="$USER_HOME/layer/snapshots"
    TEMP_DIR="$USER_HOME/tmp/layer_snapshot_extract"
    VERSION_CHECK_DIR="$USER_HOME/tmp/layerd-version-check"
    
    echo "Creating snapshot directory: $SNAPSHOT_DOWNLOAD_DIR"
    if ! mkdir -p "$SNAPSHOT_DOWNLOAD_DIR"; then
        echo "Error: Failed to create snapshot directory"
        exit 1
    fi
    
    echo "Creating temporary extraction directory: $TEMP_DIR"
    if ! mkdir -p "$TEMP_DIR"; then
        echo "Error: Failed to create temporary extraction directory"
        exit 1
    fi
    
    # Set the download path and track it for cleanup messaging
    DOWNLOADED_SNAPSHOT_FILE="$SNAPSHOT_DOWNLOAD_DIR/$SNAPSHOT_FILE"
    
    # Check if snapshot already exists (from a previous failed attempt)
    if [ -f "$DOWNLOADED_SNAPSHOT_FILE" ]; then
        echo ""
        echo "Found existing snapshot file at: $DOWNLOADED_SNAPSHOT_FILE"
        while true; do
            read -p "Use existing file instead of re-downloading? (y/n): " reuse_choice
            case "$reuse_choice" in
                y|Y|yes|Yes|YES)
                    echo "Using existing snapshot file..."
                    break
                    ;;
                n|N|no|No|NO)
                    echo "Removing existing file and downloading fresh copy..."
                    rm -f "$DOWNLOADED_SNAPSHOT_FILE"
                    # Download the snapshot
                    echo "Downloading snapshot (this may take a while, file size is ~40-75 GB)..."
                    if ! curl -L -o "$DOWNLOADED_SNAPSHOT_FILE" "https://layer-node.com/download/$SNAPSHOT_FILE"; then
                        echo "Error: Failed to download snapshot"
                        echo ""
                        echo "If a partial download exists, it has been preserved at:"
                        echo "  $DOWNLOADED_SNAPSHOT_FILE"
                        rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
                        exit 1
                    fi
                    break
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
    else
        # Download the snapshot
        echo "Downloading snapshot (this may take a while, file size is ~40-75 GB)..."
        if ! curl -L -o "$DOWNLOADED_SNAPSHOT_FILE" "https://layer-node.com/download/$SNAPSHOT_FILE"; then
            echo "Error: Failed to download snapshot"
            echo ""
            echo "If a partial download exists, it has been preserved at:"
            echo "  $DOWNLOADED_SNAPSHOT_FILE"
            echo ""
            echo "You can retry the download by running this script again,"
            echo "or manually download and use: ./install_layer.sh $NETWORK --snapshot <path>"
            rm -rf "$TEMP_DIR" "$VERSION_CHECK_DIR"
            exit 1
        fi
    fi
    
    # Extract and install the snapshot (mark as downloaded for cleanup prompts)
    extract_and_install_snapshot "$DOWNLOADED_SNAPSHOT_FILE" "$TEMP_DIR" "true"
fi

# Create systemd service file (Linux only)
if [ "$OS" == "linux" ]; then
    echo ""
    echo "================================"
    echo "    CONFIGURING COSMOVISOR..."
    echo "================================"
    echo ""
    sleep 1

    # Add environment variables to shell RC file
    echo "Adding cosmovisor environment variables to $SHELL_RC..."

    echo "export DAEMON_NAME=layerd" >> $SHELL_RC
    echo "export DAEMON_HOME=$LAYER_HOME" >> $SHELL_RC
    echo "export DAEMON_RESTART_AFTER_UPGRADE=true" >> $SHELL_RC
    echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=false" >> $SHELL_RC
    echo "export DAEMON_POLL_INTERVAL=300ms" >> $SHELL_RC
    echo "export UNSAFE_SKIP_BACKUP=true" >> $SHELL_RC
    echo "export DAEMON_PREUPGRADE_MAX_RETRIES=0" >> $SHELL_RC
    echo "Environment variables added successfully."

    # Adding binaries to cosmovisor
    echo ""
    echo "Initializing cosmovisor with layerd binary..."
    if $USER_HOME/layer/binaries/cosmovisor/cosmovisor init $USER_HOME/layer/binaries/$LAYERD_TAG/layerd; then
        echo "Cosmovisor initialized successfully."
    else
        echo "Warning: Cosmovisor initialization failed, but continuing..."
    fi
    echo ""
    echo "========================================"
    echo " CREATING EXAMPLE SYSTEMD SERVICE FILE"
    echo "========================================"
    echo ""
    sleep 1
    SERVICE_FILE_PATH="$USER_HOME/layer/layer.service"

    USERNAME=$(get_username)
    
    # Determine the service file content based on whether NODE_MONIKER is set
    if [ -z "$NODE_MONIKER" ]; then
        # No NODE_MONIKER, so don't include --key-name flag
        cat > "$SERVICE_FILE_PATH" << EOF
[Unit]
Description=Cosmovisor Layer Node Service
After=network-online.target

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=$USER_HOME/layer
ExecStart=$USER_HOME/layer/binaries/cosmovisor/cosmovisor run start --home $LAYER_HOME --keyring-backend="$KEYRING_BACKEND" --api.enable --api.swagger
Restart=always
RestartSec=10
Environment="DAEMON_NAME=layerd"
Environment="DAEMON_HOME=$LAYER_HOME"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_POLL_INTERVAL=300ms"
Environment="UNSAFE_SKIP_BACKUP=true"
Environment="DAEMON_PREUPGRADE_MAX_RETRIES=0"

[Install]
WantedBy=multi-user.target
EOF
    else
        # Include --key-name flag with NODE_MONIKER
        cat > "$SERVICE_FILE_PATH" << EOF
[Unit]
Description=Cosmovisor Layer Node Service
After=network-online.target

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=$USER_HOME/layer
ExecStart=$USER_HOME/layer/binaries/cosmovisor/cosmovisor run start --home $LAYER_HOME --keyring-backend="$KEYRING_BACKEND" --key-name="$NODE_MONIKER" --api.enable --api.swagger
Restart=always
RestartSec=10
Environment="DAEMON_NAME=layerd"
Environment="DAEMON_HOME=$LAYER_HOME"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_POLL_INTERVAL=300ms"
Environment="UNSAFE_SKIP_BACKUP=true"
Environment="DAEMON_PREUPGRADE_MAX_RETRIES=0"

[Install]
WantedBy=multi-user.target
EOF
    fi

    echo ""
    echo "================================"
    echo "SYSTEMD SERVICE FILE CREATED"
    echo "================================"
    echo ""
    echo "An example systemd service file has been created at:"
    echo "  $SERVICE_FILE_PATH"
    echo ""
    echo "To install and activate it as a system service, run:"
    echo "  sudo cp $SERVICE_FILE_PATH /etc/systemd/system/layer.service"
    echo "  sudo systemctl daemon-reload"
    echo "  sudo systemctl enable layer.service"
    echo "  sudo systemctl start layer.service"
    echo ""
    echo "To check the service status:"
    echo "  sudo systemctl status layer.service"
    echo ""
    echo "To view logs:"
    echo "  sudo journalctl -fu layer.service"
    echo ""
    echo "To start your node in a foreground terminal:"
    echo "  cd $USER_HOME/layer/binaries/$LAYERD_TAG && ./layerd start --home $LAYER_HOME --keyring-backend $KEYRING_BACKEND --api.enable --api.swagger"
    echo ""
    echo "================================"
    echo "    NODE SETUP COMPLETE :)"
    echo "================================"
    echo ""
fi

if [ "$OS" == "mac" ]; then
    echo "All done!"
    echo "Layerd start command:"
    echo ""
    echo "cd $USER_HOME/layer/binaries/$LAYERD_TAG && ./layerd start --home $LAYER_HOME --keyring-backend $KEYRING_BACKEND --api.enable --api.swagger"
    echo ""
    echo "================================"
    echo "    NODE SETUP COMPLETE :)"
    echo "================================"
    echo ""
fi
