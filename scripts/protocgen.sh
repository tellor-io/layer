#!/usr/bin/env bash

set -e

# Function to check if a command exists
check_command() {
    local cmd=$1
    if ! command -v "$cmd" &> /dev/null; then
        echo "Error: $cmd is not installed. Run 'make proto-install-deps' first."
        exit 1
    fi
}

# Verify required tools are installed
check_command buf
check_command protoc-gen-gocosmos
check_command protoc-gen-go-grpc
check_command protoc-gen-go
check_command protoc-gen-go-pulsar
check_command protoc-gen-grpc-gateway
check_command protoc-gen-openapiv2

# Directory containing proto files relative to the script's location
PROTO_DIR="./proto"
# Path to the buf template, assuming it's in the proto directory
TEMPLATE_PATH="${PROTO_DIR}/buf.gen.gogo.yaml"
# Create a temporary directory for the output
OUTPUT_DIR="."

# Find and process all proto files, excluding module.proto files
# Module config files should only get pulsar generation, not gogo
failed_files=""
while IFS= read -r -d '' proto_file; do
    # Only generate gogo proto if the file has a go_package option
    # This follows the Cosmos SDK pattern where module configs don't have go_package
    if grep -q "option go_package" "${proto_file}"; then
        echo "Generating gogo for: ${proto_file}"
        if ! buf generate --template "${TEMPLATE_PATH}" --output "${OUTPUT_DIR}" --error-format=json --log-format=json "${proto_file}"; then
            failed_files="${failed_files}${proto_file}\n"
            echo "Error: Failed to process ${proto_file}"
        fi
    else
        echo "Skipping gogo generation for ${proto_file} (no go_package option)"
    fi
done < <(find "${PROTO_DIR}" -name '*.proto' -not -path '*/module/*' -print0)

if [ -n "${failed_files}" ]; then
    echo -e "\nFailed to generate the following files:\n${failed_files}"
    exit 1
fi

# move proto files to the right places
cp -r github.com/tellor-io/layer/* ./
rm -rf github.com

./scripts/protocgen-pulsar.sh

go mod tidy

echo "Proto file generation completed successfully."
