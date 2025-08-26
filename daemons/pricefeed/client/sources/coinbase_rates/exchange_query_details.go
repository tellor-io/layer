package coinbase_rates

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

var CoinbaseRatesDetails = types.ExchangeQueryDetails{
	Exchange:      exchange_common.EXCHANGE_ID_COINBASE_RATES,
	Url:           "https://api.coinbase.com/v2/exchange-rates",
	PriceFunction: CoinbaseRatesPriceFunction,
}