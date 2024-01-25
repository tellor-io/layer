package testexchange

import (
	"fmt"

	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/coinbase_pro"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

// Exchange used for testing purposes. We'll reuse the CoinbasePro price function.
var (
	TestExchangeHost    = "test.exchange"
	TestExchangePort    = "9888"
	TestExchangeDetails = types.ExchangeQueryDetails{
		Exchange:      exchange_common.EXCHANGE_ID_TEST_EXCHANGE,
		Url:           fmt.Sprintf("http://%s:%s/ticker?symbol=$", TestExchangeHost, TestExchangePort),
		PriceFunction: coinbase_pro.CoinbaseProPriceFunction,
	}
)
