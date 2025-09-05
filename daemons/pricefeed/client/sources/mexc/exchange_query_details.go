package mexc

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var MexcDetails = types.ExchangeQueryDetails{
	Exchange:      exchange_common.EXCHANGE_ID_MEXC,
	Url:           "https://api.mexc.com/api/v3/ticker/24hr",
	PriceFunction: MexcPriceFunction,
	IsMultiMarket: true,
}
