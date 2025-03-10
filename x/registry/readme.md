# `x/registry`

## Abstract

This module enables the registration of fully customizable data specs to inform reporters . Users can vote on data specs to be altered. For more information, reference the [ADRs](#adrs) below.

## ADRs

- [adr002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr002.md) - queryId time frame structure
- [adr1004](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1004.md) - fees on tips
- [adr2002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2002.md) - nonces for bridging

## Transactions 

### RegisterSpec
Register a new data spec.
- `./layerd tx registry register-spec [query-type] [spec]`

### UpdateSpec
Update an existing data spec through governance.

### RemoveDataSpecs
Remove a data spec or specs through governance.

## Getters

### Params
-`./layerd query registry params`

### DecodeQueryData
-`./layerd query registry decode-querydata [query-data]`

### DecodeValue
-`./layerd query registry decode-value [query-type] [value]`

### GenerateQueryData
-`./layerd query registry generate-querydata [query-type] [parameters]`

### GetDataSpec
-`./layerd query registry data-spec [query-type]`

## Mocks

`make mock-gen-registry`

## Events
| Event | Handler Function |
|-------|-----------------|
| register_data_spec | RegisterSpec |
| data_spec_updated | UpdateDataSpec |


### Example Commands

```sh
# register data specification
layerd tx registry register-spec BTCBalance '{"document_hash":"<ipfs-hash>","response_value_type":"uint256","abi_components": [{"name":"address","type":"string"},{"name":"timestamp","type":"uint256"}],"aggregation_method":"weighted-mode"}' --from alice -y --chain-id layer
```

```sh
# get data spec for a registered query type
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
#   report_block_window: 0s
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
