# `x/registry`

## Abstract

This module enables the registration of fully customizable data specs to inform reporters . Users can vote on data specs to be altered. For more information, reference the [ADRs](#adrs) below.

## ADRs

- adr002 - queryId time frame structure
- adr1004 - fees on tips
- adr2002 - nonces for bridging

## Transactions 

-`RegisterSpec`
-`UpdateSpec`

## Getters

- `Params` - get module parameters
- `DecodeQueryData` - decode query data into query type and data fields
- `DecodeValue` - decode value into a string
- `GenerateQueryData` - generate query data for a given query type and data
- `GetDataSpec` - get data specification for a given query type

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands

```sh
# register data specification
layerd tx registry register-spec BTCBalance '{"document_hash":"<ipfs-hash>","response_value_type":"uint256","abi_components": [{"name":"address","type":"string"},{"name":"timestamp","type":"uint256"}],"aggregation_method":"weighted-mode"}' --from alice -y --chain-id layer
```

```sh
# get data spec for a registere query type
layerd query registry data-spec BTCBalance
# output-> 
# spec:
#   abi_components:
#   - name: address
#     type: string
#   - name: timestamp
#     type: uint256
#   aggregation_method: weighted-mode
#   document_hash: <ipfs-hash>
#   registrar: tellor13ffxsq63xsqk2a4tg94sax5mrs4jjph8hszt7v
#   report_buffer_window: 0s
#   response_value_type: uint256
```

```sh
#  generate query data
layerd query registry generate-querydata BTCBalance '["3Cyd2ExaAEoTzmLNyixJxBsJ4X16t1VePc","1705954706"]'
# output-> querydata: 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000a42544342616c616e63650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000065aecd920000000000000000000000000000000000000000000000000000000000000022334379643245786141456f547a6d4c4e7969784a7842734a34583136743156655063000000000000000000000000000000000000000000000000000000000000
```

```sh
# decode a given query data into output that matches the abi components
layerd query registry decode-querydata  00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000a42544342616c616e63650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000065aecd920000000000000000000000000000000000000000000000000000000000000022334379643245786141456f547a6d4c4e7969784a7842734a34583136743156655063000000000000000000000000000000000000000000000000000000000000
# output-> spec: 'BTCBalance: ["3Cyd2ExaAEoTzmLNyixJxBsJ4X16t1VePc",1705954706]'
```

```sh
# decode value for a given query type and hex string value
layerd query registry decode-value BTCBalance 0x00000000000000000000000000000000000000000000000000844befbf074c00
# output-> decodedValue: '[37238190000000000]'
```
