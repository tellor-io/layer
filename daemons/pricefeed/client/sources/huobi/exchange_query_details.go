package huobi

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var HuobiDetails = types.ExchangeQueryDetails{
	Exchange:      exchange_common.EXCHANGE_ID_HUOBI,
	Url:           "https://api.huobi.pro/market/tickers",
	PriceFunction: HuobiPriceFunction,
	IsMultiMarket: true,
}
