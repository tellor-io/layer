# Daemon

**Note:** Daemon services code was adopted from dydx [](https://github.com/dydxprotocol/v4-chain/tree/main/protocol/daemons) and reconfigured.

## Task loops

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

## Market Pair defaults

Defaults for market pair can be found [here](./configs/default_market_param_config.go)

example:

```go
[[market_params]]
ExchangeConfigJson = "{\"exchanges\":[{\"exchangeName\":\"Binance\",\"ticker\":\"\\\"ETHBTC\\\"\"},{\"exchangeName\":\"Bitfinex\",\"ticker\":\"tETHBTC\",\"adjustByMarket\":\"BTC-USD\"}]}" // this is just an example to show how to use adjustByMarket.  you can use ETH-USD without adjustbymarket
Exponent = -6
Id = 2
MinExchanges = 1
MinPriceChangePpm = 1000
Pair = "ETH-BTC"
QueryData = "0000.."
```

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
    // Query data is the market pair represention in layer
    QueryData string
}
```

**Note:**
A price is valid by default up to 30 seconds; to change this to a different default edit the `constants.MaxPriceAge`

**Also:** Config files are written to homedir/.layer/config/.
To change/add exchange details or market pairs edit the files `pricefeed_exchange_config.toml` or `market_params.toml` respectively.

### Median Server

Median server was added for a way to query median values that were from an endpoint or cli. See usage [here](../x/oracle/client/cli/query_all_get_median.go).
All median values or median value given query data using the following commands respectively.
`layerd query oracle get-all-median-values`
`layerd query oracle get-median-value <querydata>`

## How to add a market pair as defaults to be queried with existing APIs [Exchange_Details](./constants/static_exchange_details.go)?

- Add market_id for your pair in [exchange_common](./exchange_common/market_id.go)

```go
const (
    BTCUSD_ID uint32 = 0
    ETHUSD_ID uint32 = 1
    TRBUSD_ID uint32 = 69
    NEWPAIR_ID uint32 = <unique-number>
)
```

- Add market param config to [static_market_params_config](./constants/static_market_params_config.go)

```go
exchange_common.TRBUSD_ID: {
        Id:                 exchange_common.TRBUSD_ID,
        Pair:               `"TRB-USD"`,
        Exponent:           -6,
        MinExchanges:       1,
        MinPriceChangePpm:  1000,
        ExchangeConfigJson: `{\"exchanges\":[{\"exchangeName\":\"Binance\",\"ticker\":\"\\\"TRBUSDT\\\"\"},{\"exchangeName\":\"Bybit\",\"ticker\":\"TRBUSDT\"},{\"exchangeName\":\"CoinbasePro\",\"ticker\":\"TRB-USD\"}]}`,
        QueryData:          `"00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"`,
    },
```
