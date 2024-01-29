package kraken

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var (
	KrakenDetails = types.ExchangeQueryDetails{
		Exchange:      exchange_common.EXCHANGE_ID_KRAKEN,
		Url:           "https://api.kraken.com/0/public/Ticker",
		PriceFunction: KrakenPriceFunction,
		IsMultiMarket: true,
	}
)
