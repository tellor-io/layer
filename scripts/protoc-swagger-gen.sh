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
wget https://raw.githubusercontent.com/cosmos/cosmos-sdk/refs/tags/v0.50.9/proto/cosmos/gov/v1beta1/query.proto -O ./layer/tmp/gov/query.proto

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


