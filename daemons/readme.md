# Daemon gRPC Server

## Overview

Implements a gRPC server for daemon processes using Unix Domain Sockets (UDS) for communication.

## Usage

### Starting the Server

```go
server := NewServer(logger, grpcServer, fileHandler, socketAddress)
server.Start()
```

### Stopping the server

```go
server.Stop()
```

### Task loops

## PriceFetcher

- Will query exchanges for prices once or multiple times based on wether the api supports single vs multi markets; ie wether an api needs to be queried for each pair individually or can return multiple pairs at once, [See here for exchange details](./constants/static_exchange_details.go).

## PriceEncoder

- Will update cache with the queried prices and encode appropriately also make adjustments as necessary based on if adjustByMarket is defined.

### Configuration

## Exchange Config default

```go
[[exchanges]]
ExchangeId = "Binance"  // exchange identifier
IntervalMs = 2500  // Delays between sending api requests
TimeoutMs = 3000  // Max timeout
MaxQueries = 1  // Max number of calls in a loop.
```

Defaults for exchange information can be found [here](./configs/default_pricefeed_exchange_config.go)

## Market Pair defauls

```go
[[market_params]]
ExchangeConfigJson = "{\"exchanges\":[{\"exchangeName\":\"Binance\",\"ticker\":\"\\\"ETHBTC\\\"\"},{\"exchangeName\":\"Bitfinex\",\"ticker\":\"tETHBTC\",\"adjustByMarket\":\"BTC-USD\"}]}"
Exponent = -6
Id = 2
MinExchanges = 1
MinPriceChangePpm = 1000
Pair = "ETH-BTC"
```

Defaults for market pair can be found [here](./configs/default_market_param_config.go)

```go
type MarketParam struct {
    // Unique, sequentially-generated value.
    Id uint32
    // The human-readable name of the market pair (e.g. `BTC-USD`).
    Pair string
    // Static value. The exponent of the price.
    // For example if `Exponent == -5` then a `Value` of `1,000,000,000`
    // represents “$10,000`. Therefore `10 ^ Exponent` represents the smallest
    // price step (in dollars) that can be recorded.
    Exponent int32
    // The minimum number of exchanges that should be reporting a live price for
    // a price update to be considered valid.
    MinExchanges uint32
    // The minimum allowable change in `price` value that would cause a price
    // update on the network. Measured as `1e-6` (parts per million).
    MinPriceChangePpm uint32
    // A string of json that encodes the configuration for resolving the price
    // of this market on various exchanges.
    ExchangeConfigJson string
}
```

Note: Price Daemon is enabled by default to disable set `--price-daemon-enabled=false`
TODO: Add mapping of queryData to Id
