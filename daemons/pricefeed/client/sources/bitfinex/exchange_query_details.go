package bitfinex

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var (
	BitfinexDetails = types.ExchangeQueryDetails{
		Exchange:      exchange_common.EXCHANGE_ID_BITFINEX,
		Url:           "https://api-pub.bitfinex.com/v2/tickers?symbols=ALL",
		PriceFunction: BitfinexPriceFunction,
		IsMultiMarket: true,
	}
)
