package constants

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var (
	StaticMarketParamsConfig = map[uint32]*types.MarketParam{
		exchange_common.BTCUSD_ID: {
			Id:                 exchange_common.BTCUSD_ID,
			Pair:               `"BTC-USD"`,
			Exponent:           -5,
			MinExchanges:       1,
			MinPriceChangePpm:  1000,
			ExchangeConfigJson: `{\"exchanges\":[{\"exchangeName\":\"Binance\",\"ticker\":\"\\\"BTCUSDT\\\"\"},{\"exchangeName\":\"BinanceUS\",\"ticker\":\"\\\"BTCUSD\\\"\"},{\"exchangeName\":\"Bitfinex\",\"ticker\":\"tBTCUSD\"},{\"exchangeName\":\"Bitstamp\",\"ticker\":\"BTC/USD\"},{\"exchangeName\":\"Bybit\",\"ticker\":\"BTCUSDT\"},{\"exchangeName\":\"CoinbasePro\",\"ticker\":\"BTC-USD\"},{\"exchangeName\":\"CryptoCom\",\"ticker\":\"BTC_USD\"},{\"exchangeName\":\"Kraken\",\"ticker\":\"XXBTZUSD\"},{\"exchangeName\":\"Okx\",\"ticker\":\"BTC-USDT\"}]}`,
		},
		exchange_common.ETHUSD_ID: {
			Id:                 exchange_common.ETHUSD_ID,
			Pair:               `"ETH-USD"`,
			Exponent:           -6,
			MinExchanges:       1,
			MinPriceChangePpm:  1000,
			ExchangeConfigJson: `{\"exchanges\":[{\"exchangeName\":\"Binance\",\"ticker\":\"\\\"ETHUSDT\\\"\"},{\"exchangeName\":\"BinanceUS\",\"ticker\":\"\\\"ETHUSD\\\"\"},{\"exchangeName\":\"Bitfinex\",\"ticker\":\"tETHUSD\"},{\"exchangeName\":\"Bitstamp\",\"ticker\":\"ETH/USD\"},{\"exchangeName\":\"Bybit\",\"ticker\":\"ETHUSDT\"},{\"exchangeName\":\"CoinbasePro\",\"ticker\":\"ETH-USD\"},{\"exchangeName\":\"CryptoCom\",\"ticker\":\"ETH_USD\"},{\"exchangeName\":\"Kraken\",\"ticker\":\"XETHZUSD\"},{\"exchangeName\":\"Okx\",\"ticker\":\"ETH-USDT\"}]}`,
		},
	}
)
