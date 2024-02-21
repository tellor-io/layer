package coingecko

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var (
	CoingeckoDetails = types.ExchangeQueryDetails{
		Exchange:      exchange_common.EXCHANGE_ID_COINGECKO,
		Url:           "https://api.coingecko.com/api/v3/simple/price?ids=$&vs_currencies=usd",
		PriceFunction: CoingeckoPriceFunction,
		IsMultiMarket: true,
	}
)
