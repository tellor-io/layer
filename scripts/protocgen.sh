#!/usr/bin/env bash

set -e

GOSUMDB=off GOTOOLCHAIN=local+path go mod tidy

# Function to check if a command exists and install it if it does not
ensure_command() {
    local cmd=$1
    local package=$2

    if ! command -v "$cmd" &> /dev/null; then
        echo "$cmd could not be found, installing..."
        go install "$package"
    fi
}

# Ensure buf and other required tools are installed
ensure_command buf github.com/bufbuild/buf/cmd/buf@latest
ensure_command protoc-gen-gocosmos github.com/cosmos/gogoproto/protoc-gen-gocosmos@latest
ensure_command protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ensure_command protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go@latest
ensure_command protoc-gen-go-pulsar github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest
ensure_command protoc-gen-grpc-gateway github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@latest
ensure_command protoc-gen-openapiv2 github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
ensure_command goimports golang.org/x/tools/cmd/goimports@latest

# Directory containing proto files relative to the script's location
PROTO_DIR="./proto"
# Path to the buf template, assuming it's in the proto directory
TEMPLATE_PATH="${PROTO_DIR}/buf.gen.gogo.yaml"
# Create a temporary directory for the output
OUTPUT_DIR="."

# Find and process all proto files
find "${PROTO_DIR}" -name '*.proto' -print0 | while IFS= read -r -d '' proto_file; do
    buf generate --template "${TEMPLATE_PATH}" --output "${OUTPUT_DIR}" --error-format=json --log-format=json "${proto_file}"
    if [ $? -ne 0 ]; then
        echo "Failed to process ${proto_file}"
        exit 1
    fi
done

# move proto files to the right places
cp -r github.com/tellor-io/layer/* ./
rm -rf github.com

./scripts/protocgen-pulsar.sh

go mod tidy

echo "Proto file generation completed successfully."
