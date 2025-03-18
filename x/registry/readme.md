# `x/registry`

## Abstract

This module enables the registration of fully customizable data specs to inform reporters . Users can vote on data specs to be altered. For more information, reference the [ADRs](#adrs) below.

## ADRs

- [adr002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr002.md) - queryId time frame structure
- [adr1004](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1004.md) - fees on tips
- [adr2002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2002.md) - nonces for bridging

## How to add new queries to Layer:
Before registering new queries, please check the existing specs using:  
```./layerd query registry all-data-specs```  

The registry module uses the [`RegisterSpec`](https://github.com/tellor-io/layer/blob/main/x/registry/keeper/msg_server_register_spec.go#L19) function to register new queries to Layer.

 The `RegisterSpec` function takes in a string queryType and a [data spec](https://github.com/tellor-io/layer/blob/main/x/registry/types/data_spec.pb.go#L92). The queryType is the title of the data spec, such as "spotprice", and the data spec fields describe the queryType. A dataspec contains the following fields:
```json
{
  "document_hash": "your-ipfs-hash",
  "response_value_type": "string",
  "aggregation_method": "weighted-mode",
  "abi_components": [
    {
      "name": "parameter1",
      "field_type": "string",
    }
  ],
  "registrar": "your-wallet-address",
  "query_type": "YourQueryType",
  "report_block_window": 10
}
```


When building the tx, make sure your dataspec json is minified (single line), and that it follows the requirements in [`RegisterSpec`](https://github.com/tellor-io/layer/blob/main/x/registry/keeper/msg_server_register_spec.go#L19). For example, if you wanted to register a new queryType called "NFLSuperBowlChampion", the data spec could look like:  
```bash 
{"document_hash":"your-ipfs-hash","response_value_type":"string","abi_components":[{"name":"year game was played","field_type":"string"}],"aggregation_method":"weighted-mode","registrar":"your-layer-address","report_block_window":10,"query_type":"NFLSuperBowlChampion"}
```


 An example tx registering an NFLSuperBowlChampion query:

```bash
layerd tx registry register-spec NFLSuperBowlChampion {\"document_hash\":\"legit-ipfs-hash!\",\"response_value_type\":\"string\",\"abi_components\":[{\"name\":\"year of game\",\"field_type\":\"string\"}],\"aggregation_method\":\"weighted-mode\",\"registrar\":\"tellor1nsjmvmkfmgpx3g7j2aw7m5d6pr2fxp6dqfglu6\",\"report_block_window\":10,\"query_type\":\"NFLSuperBowlChampion\"} --keyring-dir /var/cosmos-chain/layer-1 --gas 1000 --fees 10loya --from tellor1nsjmvmkfmgpx3g7j2aw7m5d6pr2fxp6dqfglu6 --keyring-backend test --output json -y --chain-id layer
```

(A test manually registering, tipping, and reporting this query can be found [here](https://github.com/tellor-io/layer/blob/5820469f2544b2dc1a34ac06b961b92a4adcb782/e2e/dispute_test.go#L2459))

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
- `./layerd query registry params`

### DecodeQueryData
- `./layerd query registry decode-querydata [query-data]`

### DecodeValue
- `./layerd query registry decode-value [query-type] [value]`

### GenerateQueryData
- `./layerd query registry generate-querydata [query-type] [parameters]`

### GetDataSpec
- `./layerd query registry data-spec [query-type]`

### GetAllDataSpecs
- `./layerd query registry all-data-specs`

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
#   query_type: btcbalance
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
