# !/usr/bin/env bash

# Directory containing proto files relative to the script's location
PROTO_DIR="./proto"
OUTPUT_DIR="."
# Path to the buf template, assuming it's in the proto directory
TEMPLATE_PATH="${PROTO_DIR}/buf.gen.pulsar.yaml"

# Find and process all proto files
find "${PROTO_DIR}" -name '*.proto' -print0 | while IFS= read -r -d '' proto_file; do
    buf generate --template "${TEMPLATE_PATH}" --output "${OUTPUT_DIR}" --error-format=json --log-format=json "${proto_file}"
    if [ $? -ne 0 ]; then
        echo "Failed to process ${proto_file}"
        exit 1
    fi
done

if [ -d "layer" ]; then
    cp -r layer/* ./api
    rm -rf layer
fi
