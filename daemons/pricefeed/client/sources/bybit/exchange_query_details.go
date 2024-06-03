package bybit

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var BybitDetails = types.ExchangeQueryDetails{
	Exchange:      exchange_common.EXCHANGE_ID_BYBIT,
	Url:           "https://api.bybit.com/v5/market/tickers?category=spot",
	PriceFunction: BybitPriceFunction,
	IsMultiMarket: true,
}
