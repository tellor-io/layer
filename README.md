# Tellor Layer<br/><br/>

<p align="center">
  <a href="https://github.com/tellor-io/layer/actions/workflows/go.yml">
    <img src="https://github.com/tellor-io/layer/actions/workflows/go.yml/badge.svg" alt="Tests" />
  </a>
  <a href='https://twitter.com/WeAreTellor'>
    <img src='https://img.shields.io/twitter/url/http/shields.io.svg?style=social' alt='Twitter WeAreTellor' />
  </a>
</p>

## Overview <a name="overview"> </a>

<b>Tellor Layer</b> is a stand alone L1 built using the cosmos sdk for the purpose of coming to
consensus on any subjective data. It works by using a network of staked parties who are
crypto-economically incentivized to honestly report requested data.

For more in-depth information, checkout the [Tellor Layer tech paper](https://github.com/tellor-io/layer/blob/main/TellorLayer%20-%20tech.pdf) and our [ADRs](https://github.com/tellor-io/layer/tree/main/adr).

For docs on how to join our public testnet go here:  [https://docs.tellor.io/layer-docs](https://docs.tellor.io/layer-docs)

## Starting a New Chain

1) Select the start script that works for you

- `start_one_node.sh` is for those who want to run a chain with a single validator in a mac environment
- `start_one_node_aws.sh` is for those who want a chain with a single validator and the option to import a faucet account from a seed phrase to be used in a linux environment
- `start_two_chains.sh` (mac environment) sets up two nodes/validators and starts one of them from this script. Then to start the other validator you would run the `start_bill.sh` script

2) Run the selected script from the base layer folder:

```sh
./start_scripts/{selected_script}
```

## Joining a Running Chain

To find more information please go to the layer_scripts folder.

Here you will find a detailed breakdown for how to join a chain as a node and how to create a new validator for the chain

## Start a local devnet

Run the chain locally in a docker container, powered by [local-ic](https://github.com/strangelove-ventures/interchaintest/tree/main/local-interchain)

Install heighliner:
```sh
make get-heighliner
```
Create image:
```sh
make local-image
```
Install local interchain:
```sh
make get-localic
```
Start the local-devnet:
```sh
make local-devnet
```

To configure the chain (ie add more validators plus more) edit the json in local_devnet/chains/layer.json


## Tests

To run all unit and integration tests:

```go
make test
```

In addition to the unit and integration tests, there are also end to end tests using the [interchaintest](https://github.com/strangelove-ventures/interchaintest) framework. These tests spin up a live chain with a given number of nodes/validators in docker that you can run transactions and queries against. To run all e2e tests:

Install heighliner:
```sh
make get-heighliner
```
Create image:
```sh
make local-image
```
Run all e2e tests:
```sh
make e2e
```
Run an individual test:
```sh
cd e2e
go test -v -run TestLayerFlow -timeout 10m
```
Run benchmark tests:
```sh
make bench-report
```


## Linting

To lint per folder:  
`make lint-folder-fix FOLDER="x/mint"`

To lint all files:

`make lint`

## Adding New Getters/Queries
Read about queries from the Cosmos SDK team [here](https://github.com/cosmos/cosmos-sdk/blob/267b93ee741144c0e6a3d57840a006761d07e6c3/docs/learn/beginner/02-query-lifecycle.md). All nodes do not have to have the same queries. Feel free to make an issue or PR regarding any desired queries to this repo. To add a new query to layer, replace the logic as needed for [`GetCurrentAggregateReport`](https://github.com/tellor-io/layer/blob/5820469f2544b2dc1a34ac06b961b92a4adcb782/x/oracle/keeper/query.go#L46):

1. In `layer/proto/layer/oracle/query.go` , define the rpc endpoint in the `service Query {}` struct. 
```proto
service Query {
  ...
  // Queries the current aggregate report by query id
  rpc GetCurrentAggregateReport(QueryGetCurrentAggregateReportRequest) returns (QueryGetCurrentAggregateReportResponse) {
    option (google.api.http).get = "/tellor-io/layer/oracle/get_current_aggregate_report/{query_id}";
  }
  ...
}
```
2. Also in `layer/proto/layer/oracle/query.go`, define a Message for the query request and query response. 

```proto
// QueryGetCurrentAggregateReportRequest is the request type for the Query/GetCurrentAggregateReport RPC method.
message QueryGetCurrentAggregateReportRequest {
  // query_id defines the query id hex string.
  string query_id = 1;
}
// QueryGetCurrentAggregateReportResponse is the response type for the Query/GetCurrentAggregateReport RPC method.
message QueryGetCurrentAggregateReportResponse {
  // aggregate defines the current aggregate report.
  Aggregate aggregate = 1;
  // timestamp defines the timestamp of the aggregate report.
  uint64 timestamp = 2;
}
```

**Note**: When inputting or returning a list of items, use the `repeated` keyword, and optional pagination. Ex. with a different query [here](https://github.com/tellor-io/layer/blob/5820469f2544b2dc1a34ac06b961b92a4adcb782/proto/layer/oracle/query.proto#L329)  
**Note**: If inputting or returning a custom type, such as `Aggregate`, it needs to be defined and imported within the proto files. Ex [here](https://github.com/tellor-io/layer/blob/5820469f2544b2dc1a34ac06b961b92a4adcb782/proto/layer/oracle/aggregate.proto#L11)

3. Generate the protobuf files for the new query using a custom protoc implementation or `ignite generate proto-go`. (Ignite docs [here](https://github.com/ignite/cli).)

4. Create the query function in layer/x/oracle/keeper/query.go or individual query file. 
```go
// gets the current aggregate report for a query id
func (k Querier) GetCurrentAggregateReport(ctx context.Context, req *types.QueryGetCurrentAggregateReportRequest) (*types.QueryGetCurrentAggregateReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	aggregate, timestamp, err := k.keeper.GetCurrentAggregateReport(ctx, queryId)
	if err != nil {
		return nil, err
	}
	timeUnix := timestamp.UnixMilli()

	return &types.QueryGetCurrentAggregateReportResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
```

5. Add test(s) 

6. In `layer/x/oracle/autocli.go`, add your query to the `AutoCLIOptions` return  

```go
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
        ...
				{
					RpcMethod:      "GetCurrentAggregateReport",
					Use:            "get-current-aggregate-report [query_id]",
					Short:          "Query current aggregate report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
        ...
      }
    }
  }
  ...
}
```

7. Run `./scripts/protoc-swagger-gen.sh ` to generate the swagger page documentation

### Using pagination
Some queries return very long lists from storage. To only retrieve what you need, you can use pagination. 
At the end of a query command, you can add:
- `--page-limit n` to only return n results
- `--page-reverse` to retreive the keys in descending order
- `--page-count-total` to see the total number of results returned

For example, if you wanted to only return the most recent 10 reports for a certain reporter, you can use 
```bash
./layerd query oracle get-reportsby-reporter $REP_ADDR --page-limit 10 --page-reverse
```

## Maintainers<a name="maintainers"> </a>

This repository is maintained by the [Tellor team](https://github.com/orgs/tellor-io/people)

## How to Contribute<a name="how2contribute"> </a>  

Check out our issues log here on Github or feel free to reach out anytime [info@tellor.io](mailto:info@tellor.io)

## Community<a name="community"> </a>  

- [Official Website](https://tellor.io/)
- [Discord](https://discord.gg/n7drGjh)
- [Twitter](https://twitter.com/wearetellor)

## Copyright<a name="copyright"> </a>  

Tellor Inc. 2025
