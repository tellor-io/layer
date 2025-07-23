#!/usr/bin/env bash

set -eo pipefail


mkdir -p ./tmp-swagger-gen
cd proto

mkdir -p ./layer/tmp/bank
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/bank/v1beta1/query.proto -O ./layer/tmp/bank/query.proto

mkdir ./layer/tmp/evidence
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/evidence/v1beta1/query.proto -O ./layer/tmp/evidence/query.proto

mkdir ./layer/tmp/staking
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/staking/v1beta1/query.proto -O ./layer/tmp/staking/query.proto

mkdir ./layer/tmp/distribution
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/distribution/v1beta1/query.proto -O ./layer/tmp/distribution/query.proto

mkdir ./layer/tmp/auth
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/auth/v1beta1/query.proto -O ./layer/tmp/auth/query.proto

mkdir ./layer/tmp/upgrade
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/upgrade/v1beta1/query.proto -O ./layer/tmp/upgrade/query.proto

mkdir ./layer/tmp/slashing
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/slashing/v1beta1/query.proto -O ./layer/tmp/slashing/query.proto

mkdir ./layer/tmp/gov
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/gov/v1/query.proto -O ./layer/tmp/gov/query.proto

mkdir ./layer/tmp/feegrant
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/feegrant/v1beta1/query.proto -O ./layer/tmp/feegrant/query.proto

mkdir ./layer/tmp/group
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/group/v1/query.proto -O ./layer/tmp/group/query.proto

mkdir ./layer/tmp/consensus
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/consensus/v1/query.proto -O ./layer/tmp/consensus/query.proto

mkdir ./layer/tmp/ibc
wget https://raw.githubusercontent.com/cosmos/ibc-go/refs/tags/v8.0.0/proto/ibc/applications/transfer/v1/query.proto -O ./layer/tmp/ibc/query.proto

mkdir ./layer/tmp/icacontroller
wget https://raw.githubusercontent.com/cosmos/ibc-go/refs/tags/v8.0.0/proto/ibc/applications/interchain_accounts/controller/v1/query.proto -O ./layer/tmp/icacontroller/query.proto

mkdir ./layer/tmp/icahost
wget https://raw.githubusercontent.com/cosmos/ibc-go/refs/tags/v8.0.0/proto/ibc/applications/interchain_accounts/host/v1/query.proto -O ./layer/tmp/icahost/query.proto

mkdir ./layer/tmp/globalfee
wget https://raw.githubusercontent.com/strangelove-ventures/globalfee/refs/tags/v0.50.1/proto/gaia/globalfee/v1beta1/query.proto -O ./layer/tmp/globalfee/query.proto

proto_dirs=$(find ./layer -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  echo $query_file
  if [[ ! -z "$query_file" ]]; then
    buf generate --template buf.gen.swagger.yaml $query_file
  fi
done

rm -rf ./layer/tmp

cd ..
# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./docs/config.json -o ./docs/static/openapi.yml -f yaml --continueOnConflictingPaths true

# clean swagger files
rm -rf ./tmp-swagger-gen

# Add snippet to openapi.yml after swagger generation

# The path to the openapi.yml file
OPENAPI_FILE="./docs/static/openapi.yml"

# Create a temporary file for our addition
TMP_FILE=$(mktemp)

# Find the line number where /layer/registry/params: occurs
LINE_NUMBER=$(grep -n "/layer/registry/params:" "$OPENAPI_FILE" | cut -d':' -f1)

# Split the file at the target line
head -n $((LINE_NUMBER-1)) "$OPENAPI_FILE" > "$TMP_FILE"

# Add our snippet
cat << 'EOF' >> "$TMP_FILE"
  /layer/registry/get_all_data_specs:
    get:
      summary: Queries a list of GetAllDataSpecs items.
      operationId: GetAllDataSpecs
      responses:
        '200':
          description: A successful response.
          schema:
            type: object
            properties:
              specs:
                type: array
                items:
                  type: object
                  properties:
                    document_hash:
                      type: string
                      title: ipfs hash of the data spec
                    response_value_type:
                      type: string
                      title: the value's datatype for decoding the value
                    abi_components:
                      type: array
                      items:
                        type: object
                        properties:
                          name:
                            type: string
                            title: name
                          field_type:
                            type: string
                            title: type
                          nested_component:
                            type: array
                            items:
                              type: object
                              properties:
                                name:
                                  type: string
                                field_type:
                                  type: string
                            title: >-
                              consider taking this recursion out and make it
                              once only
                        title: >-
                          ABIComponent is a specification for how to interpret
                          abi_components
                      title: the abi components for decoding
                    aggregation_method:
                      type: string
                      title: >-
                        how to aggregate the data (ie. average, median, mode,
                        etc) for aggregating reports and arriving at final value
                    registrar:
                      type: string
                      title: address that originally registered the data spec
                    report_block_window:
                      type: string
                      format: uint64
                      title: >-
                        report_buffer_window specifies the duration of the time
                        window following an initial report

                        during which additional reports can be submitted. This
                        duration acts as a buffer, allowing

                        a collection of related reports in a defined time frame.
                        The window ensures that all

                        pertinent reports are aggregated together before
                        arriving at a final value. This defaults

                        to 0s if not specified.

                        extensions: treat as a golang time.duration, don't allow
                        nil values, don't omit empty values
                    query_type:
                      type: string
                      title: querytype is the first arg in queryData
                  title: >-
                    DataSpec is a specification for how to interpret and
                    aggregate data
                description: specs is the list of all data specs.
            description: >-
              QueryGetAllDataSpecsResponse is response type for the
              Query/GetAllDataSpecs RPC method.
        default:
          description: An unexpected error response.
          schema:
            type: object
            properties:
              code:
                type: integer
                format: int32
              message:
                type: string
              details:
                type: array
                items:
                  type: object
                  properties:
                    '@type':
                      type: string
                  additionalProperties: {}
      tags:
        - Query
  /layer/registry/get_data_spec/{query_type}:
    get:
      summary: Queries a list of GetDataSpec items.
      operationId: GetDataSpec
      responses:
        '200':
          description: A successful response.
          schema:
            type: object
            properties:
              spec:
                description: spec is the data spec corresponding to the query type.
                type: object
                properties:
                  document_hash:
                    type: string
                    title: ipfs hash of the data spec
                  response_value_type:
                    type: string
                    title: the value's datatype for decoding the value
                  abi_components:
                    type: array
                    items:
                      type: object
                      properties:
                        name:
                          type: string
                          title: name
                        field_type:
                          type: string
                          title: type
                        nested_component:
                          type: array
                          items:
                            type: object
                            properties:
                              name:
                                type: string
                              field_type:
                                type: string
                          title: >-
                            consider taking this recursion out and make it once
                            only
                      title: >-
                        ABIComponent is a specification for how to interpret
                        abi_components
                    title: the abi components for decoding
                  aggregation_method:
                    type: string
                    title: >-
                      how to aggregate the data (ie. average, median, mode, etc)
                      for aggregating reports and arriving at final value
                  registrar:
                    type: string
                    title: address that originally registered the data spec
                  report_block_window:
                    type: string
                    format: uint64
                    title: >-
                      report_buffer_window specifies the duration of the time
                      window following an initial report

                      during which additional reports can be submitted. This
                      duration acts as a buffer, allowing

                      a collection of related reports in a defined time frame.
                      The window ensures that all

                      pertinent reports are aggregated together before arriving
                      at a final value. This defaults

                      to 0s if not specified.

                      extensions: treat as a golang time.duration, don't allow
                      nil values, don't omit empty values
                  query_type:
                    type: string
                    title: querytype is the first arg in queryData
                title: >-
                  DataSpec is a specification for how to interpret and aggregate
                  data
            description: >-
              QueryGetDataSpecResponse is response type for the
              Query/GetDataSpec RPC method.
        default:
          description: An unexpected error response.
          schema:
            type: object
            properties:
              code:
                type: integer
                format: int32
              message:
                type: string
              details:
                type: array
                items:
                  type: object
                  properties:
                    '@type':
                      type: string
                  additionalProperties: {}
      parameters:
        - name: query_type
          description: queryType is the key to fetch a the corresponding data spec.
          in: path
          required: true
          type: string
      tags:
        - Query
EOF

# Add the remaining lines including the params section
tail -n +$LINE_NUMBER "$OPENAPI_FILE" >> "$TMP_FILE"

# Replace the original file with the modified one
mv "$TMP_FILE" "$OPENAPI_FILE"

echo "API snippet for /layer/registry/get_data_spec/ and /layer/registry/get_all_data_specs added successfully to $OPENAPI_FILE"