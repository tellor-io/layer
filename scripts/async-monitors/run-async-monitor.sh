#!/bin/bash

# Async Monitor Events Runner
# This script provides a convenient way to run the async monitor with common configurations

set -e

# Default values
RPC_URL="127.0.0.1:26657"
CONFIG_FILE=""
NODE_NAME=""
BLOCK_TIME_THRESHOLD=""
TIMESTAMP_ANALYZER=""

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -r, --rpc-url URL           RPC URL (default: 127.0.0.1:26657)"
    echo "  -c, --config FILE           Path to config file (required)"
    echo "  -n, --node NAME             Name of the node being monitored (required)"
    echo "  -t, --block-time DURATION   Block time threshold (e.g., 5m, 1h)"
    echo "  -a, --timestamp-analyzer    Enable timestamp analyzer"
    echo "  -h, --help                  Display this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -c ../monitors/event-config.yml -n my-node"
    echo "  $0 -c ../monitors/event-config.yml -n my-node -t 5m -a"
    echo "  $0 --config ../monitors/event-config.yml --node my-node --block-time 10m"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--rpc-url)
            RPC_URL="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        -n|--node)
            NODE_NAME="$2"
            shift 2
            ;;
        -t|--block-time)
            BLOCK_TIME_THRESHOLD="$2"
            shift 2
            ;;
        -a|--timestamp-analyzer)
            TIMESTAMP_ANALYZER="-timestamp-analyzer"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$CONFIG_FILE" ]]; then
    echo "Error: Config file is required"
    usage
    exit 1
fi

if [[ -z "$NODE_NAME" ]]; then
    echo "Error: Node name is required"
    usage
    exit 1
fi

# Check if config file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: Config file '$CONFIG_FILE' not found"
    exit 1
fi

# Build command
CMD="go run async-monitor-events.go -rpc-url=$RPC_URL -config=$CONFIG_FILE -node=$NODE_NAME"

# Add optional parameters
if [[ -n "$BLOCK_TIME_THRESHOLD" ]]; then
    CMD="$CMD -block-time-threshold=$BLOCK_TIME_THRESHOLD"
fi

if [[ -n "$TIMESTAMP_ANALYZER" ]]; then
    CMD="$CMD $TIMESTAMP_ANALYZER"
fi

echo "Starting Async Monitor Events..."
echo "RPC URL: $RPC_URL"
echo "Config file: $CONFIG_FILE"
echo "Node name: $NODE_NAME"
if [[ -n "$BLOCK_TIME_THRESHOLD" ]]; then
    echo "Block time threshold: $BLOCK_TIME_THRESHOLD"
fi
if [[ -n "$TIMESTAMP_ANALYZER" ]]; then
    echo "Timestamp analyzer: enabled"
fi
echo ""

# Run the command
exec $CMD 