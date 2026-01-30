#!/bin/bash

# query-all-params.sh queries all module parameters from a Layer chain node
# and writes them to a single JSON file for snapshot purposes.
#
# Usage:
#   ./scripts/query-all-params.sh [layerd_path] [output_file]
#
#   Or set LAYERD environment variable:
#   LAYERD=/path/to/layerd ./scripts/query-all-params.sh [output_file]
#
# Examples:
#   ./scripts/query-all-params.sh ./layerd params-snapshot.json
#   ./scripts/query-all-params.sh /usr/local/bin/layerd
#   LAYERD=./layerd ./scripts/query-all-params.sh params-snapshot.json
#
# The output JSON file contains all module parameters with a timestamp.

# Determine layerd path from argument, environment variable, or default
if [ -n "$1" ] && [ -f "$1" ] && [ -x "$1" ]; then
    # First argument is a valid executable file - treat as layerd path
    LAYERD_CMD="$1"
    OUTPUT_FILE="${2:-all-module-params.json}"
elif [ -n "$LAYERD" ] && [ -f "$LAYERD" ] && [ -x "$LAYERD" ]; then
    # LAYERD environment variable is set and valid
    LAYERD_CMD="$LAYERD"
    OUTPUT_FILE="${1:-all-module-params.json}"
elif [ -f "./layerd" ] && [ -x "./layerd" ]; then
    # Try ./layerd in current directory
    LAYERD_CMD="./layerd"
    OUTPUT_FILE="${1:-all-module-params.json}"
elif command -v layerd &> /dev/null; then
    # Fall back to layerd in PATH
    LAYERD_CMD="layerd"
    OUTPUT_FILE="${1:-all-module-params.json}"
else
    echo "Error: layerd not found"
    echo ""
    echo "Usage:"
    echo "  $0 [layerd_path] [output_file]"
    echo "  or"
    echo "  LAYERD=/path/to/layerd $0 [output_file]"
    echo ""
    echo "Examples:"
    echo "  $0 ./layerd params-snapshot.json"
    echo "  $0 /usr/local/bin/layerd"
    echo "  LAYERD=./layerd $0 params-snapshot.json"
    exit 1
fi

# Validate layerd is executable
if [ ! -x "$LAYERD_CMD" ] && ! command -v "$LAYERD_CMD" &> /dev/null; then
    echo "Error: $LAYERD_CMD is not executable"
    exit 1
fi

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Create temporary file for building JSON
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

echo "Querying module parameters..."
echo "Using layerd: $LAYERD_CMD"
echo "Output will be written to: $OUTPUT_FILE"
echo ""

# Start building JSON structure
echo "{" > "$TEMP_FILE"
echo "  \"timestamp\": \"$TIMESTAMP\"," >> "$TEMP_FILE"
echo "  \"module_params\": {" >> "$TEMP_FILE"

FIRST=true
ERRORS=0

# Function to query a module's params
query_module() {
    local module_name=$1
    local query_cmd=$2
    
    if [ "$FIRST" = false ]; then
        echo "," >> "$TEMP_FILE"
    fi
    FIRST=false
    
    echo -n "  Querying $module_name... "
    
    # Try to query the module
    if result=$($LAYERD_CMD query $query_cmd --output json 2>/dev/null); then
        # Extract params from response (response usually has a "params" field)
        # If jq is available, use it to extract params, otherwise use the full result
        if command -v jq &> /dev/null; then
            params=$(echo "$result" | jq -c '.params // .' 2>/dev/null || echo "$result")
        else
            params="$result"
        fi
        echo "\"$module_name\": $params" >> "$TEMP_FILE"
        echo "OK"
    else
        echo "\"$module_name\": null" >> "$TEMP_FILE"
        echo "ERROR"
        ((ERRORS++))
    fi
}

# Custom Layer modules
query_module "bridge" "bridge params"
query_module "dispute" "dispute params"
query_module "oracle" "oracle params"
query_module "registry" "registry params"
query_module "reporter" "reporter params"

# Standard Cosmos SDK modules
query_module "auth" "auth params"
query_module "bank" "bank params"
query_module "staking" "staking params"
query_module "distribution" "distribution params"
query_module "slashing" "slashing params"
query_module "gov" "gov params"
query_module "consensus" "consensus params"
query_module "evidence" "evidence params"
query_module "feegrant" "feegrant params"
query_module "authz" "authz params"
query_module "group" "group params"
query_module "upgrade" "upgrade params"

# IBC modules
query_module "ibc-transfer" "ibc-transfer params"
query_module "ibc" "ibc client params"

# Global fee module
query_module "globalfee" "globalfee params"

# Close JSON structure
echo "" >> "$TEMP_FILE"
echo "  }" >> "$TEMP_FILE"
echo "}" >> "$TEMP_FILE"

# Format JSON and write to output file
if command -v jq &> /dev/null; then
    jq . "$TEMP_FILE" > "$OUTPUT_FILE"
else
    cp "$TEMP_FILE" "$OUTPUT_FILE"
    echo ""
    echo "Warning: jq not found. Output JSON is not formatted."
    echo "Install jq for formatted output: brew install jq (macOS) or apt-get install jq (Linux)"
fi

echo ""
if [ $ERRORS -eq 0 ]; then
    echo "✓ Successfully queried all modules"
else
    echo "⚠ $ERRORS modules had errors (see output file)"
fi
echo "✓ Results written to: $OUTPUT_FILE"
