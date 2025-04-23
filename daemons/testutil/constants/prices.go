package constants

import (
	pricefeedclient "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/daemons/testutil/daemons/pricefeed/exchange_config"
)

const (
	BtcUsdPair   = "BTC-USD"
	EthUsdPair   = "ETH-USD"
	MaticUsdPair = "MATIC-USD"
	SolUsdPair   = "SOL-USD"
	LtcUsdPair   = "LTC-USD"

	BtcUsdExponent   = -5
	EthUsdExponent   = -6
	LinkUsdExponent  = -8
	MaticUsdExponent = -9
	CrvUsdExponent   = -10
	SolUsdExponent   = -8
	LtcUsdExponent   = -7

	CoinbaseExchangeName  = "Coinbase"
	BinanceExchangeName   = "Binance"
	BinanceUSExchangeName = "BinanceUS"
	BitfinexExchangeName  = "Bitfinex"
	KrakenExchangeName    = "Kraken"

	FiveBillion  = uint64(5_000_000_000)
	ThreeBillion = uint64(3_000_000_000)
	FiveMillion  = uint64(5_000_000)
	OneMillion   = uint64(1_000_000)

	// Market param validation errors.
	ErrorMsgMarketPairCannotBeEmpty = "Pair cannot be empty"
	ErrorMsgInvalidMinPriceChange   = "Min price change in parts-per-million must be greater than 0 and less than 10000"
)

var TestMarketExchangeConfigs = map[pricefeedclient.MarketId]string{
	exchange_config.MARKET_BTC_USD: `{
		"exchanges": [
		  {
			"exchangeName": "Binance",
			"ticker": "BTCUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "BinanceUS",
			"ticker": "BTCUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Bitfinex",
			"ticker": "tBTCUSD"
		  },
		  {
			"exchangeName": "Bitstamp",
			"ticker": "BTC/USD"
		  },
		  {
			"exchangeName": "Bybit",
			"ticker": "BTCUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "CoinbasePro",
			"ticker": "BTC-USD"
		  },
		  {
			"exchangeName": "CryptoCom",
			"ticker": "BTC_USD"
		  },
		  {
			"exchangeName": "Kraken",
			"ticker": "XXBTZUSD"
		  },
		  {
			"exchangeName": "Mexc",
			"ticker": "BTC_USDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Okx",
			"ticker": "BTC-USDT",
			"adjustByMarket": "USDT-USD"
		  }
		]
	  }`,
	exchange_config.MARKET_ETH_USD: `{
		"exchanges": [
		  {
			"exchangeName": "Binance",
			"ticker": "ETHUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "BinanceUS",
			"ticker": "ETHUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Bitfinex",
			"ticker": "tETHUSD"
		  },
		  {
			"exchangeName": "Bitstamp",
			"ticker": "ETH/USD"
		  },
		  {
			"exchangeName": "Bybit",
			"ticker": "ETHUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "CoinbasePro",
			"ticker": "ETH-USD"
		  },
		  {
			"exchangeName": "CryptoCom",
			"ticker": "ETH_USD"
		  },
		  {
			"exchangeName": "Kraken",
			"ticker": "XETHZUSD"
		  },
		  {
			"exchangeName": "Mexc",
			"ticker": "ETH_USDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Okx",
			"ticker": "ETH-USDT",
			"adjustByMarket": "USDT-USD"
		  }
		]
	  }`,
	exchange_config.MARKET_SOL_USD: `{
		"exchanges": [
		  {
			"exchangeName": "Binance",
			"ticker": "SOLUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Bitfinex",
			"ticker": "tSOLUSD",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Bybit",
			"ticker": "SOLUSDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "CoinbasePro",
			"ticker": "SOL-USD"
		  },
		  {
			"exchangeName": "CryptoCom",
			"ticker": "SOL_USD"
		  },
		  {
			"exchangeName": "Huobi",
			"ticker": "solusdt",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Kraken",
			"ticker": "SOLUSD"
		  },
		  {
			"exchangeName": "Kucoin",
			"ticker": "SOL-USDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Mexc",
			"ticker": "SOL_USDT",
			"adjustByMarket": "USDT-USD"
		  },
		  {
			"exchangeName": "Okx",
			"ticker": "SOL-USDT",
			"adjustByMarket": "USDT-USD"
		  }
		]
	  }`,
}

var TestMarketParams = []pricefeedclient.MarketParam{
	{
		Id:                 0,
		Pair:               BtcUsdPair,
		Exponent:           BtcUsdExponent,
		MinExchanges:       1,
		MinPriceChangePpm:  50,
		ExchangeConfigJson: TestMarketExchangeConfigs[exchange_config.MARKET_BTC_USD],
	},
	{
		Id:                 1,
		Pair:               EthUsdPair,
		Exponent:           EthUsdExponent,
		MinExchanges:       1,
		MinPriceChangePpm:  50,
		ExchangeConfigJson: TestMarketExchangeConfigs[exchange_config.MARKET_ETH_USD],
	},
	{
		Id:                 2,
		Pair:               SolUsdPair,
		Exponent:           SolUsdExponent,
		MinExchanges:       1,
		MinPriceChangePpm:  50,
		ExchangeConfigJson: TestMarketExchangeConfigs[exchange_config.MARKET_SOL_USD],
	},
}
