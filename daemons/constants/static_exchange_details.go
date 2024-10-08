package constants

import (
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/binance"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/bitfinex"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/bitstamp"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/crypto_com"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/gate"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/huobi"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/kraken"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/kucoin"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/mexc"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/okx"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/test_fixed_price_exchange"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/test_volatile_exchange"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/testexchange"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

// StaticExchangeDetails is the static mapping of `ExchangeId` to its `ExchangeQueryDetails`.
var StaticExchangeDetails = map[types.ExchangeId]types.ExchangeQueryDetails{
	exchange_common.EXCHANGE_ID_BINANCE:    binance.BinanceDetails,
	exchange_common.EXCHANGE_ID_BINANCE_US: binance.BinanceUSDetails,
	exchange_common.EXCHANGE_ID_BITFINEX:   bitfinex.BitfinexDetails,
	exchange_common.EXCHANGE_ID_KRAKEN:     kraken.KrakenDetails,
	exchange_common.EXCHANGE_ID_GATE:       gate.GateDetails,
	exchange_common.EXCHANGE_ID_BITSTAMP:   bitstamp.BitstampDetails,
	// exchange_common.EXCHANGE_ID_BYBIT:                     bybit.BybitDetails,
	exchange_common.EXCHANGE_ID_CRYPTO_COM: crypto_com.CryptoComDetails,
	exchange_common.EXCHANGE_ID_HUOBI:      huobi.HuobiDetails,
	exchange_common.EXCHANGE_ID_KUCOIN:     kucoin.KucoinDetails,
	exchange_common.EXCHANGE_ID_OKX:        okx.OkxDetails,
	exchange_common.EXCHANGE_ID_MEXC:       mexc.MexcDetails,
	// exchange_common.EXCHANGE_ID_COINBASE_PRO:              coinbase_pro.CoinbaseProDetails,
	exchange_common.EXCHANGE_ID_TEST_EXCHANGE:             testexchange.TestExchangeDetails,
	exchange_common.EXCHANGE_ID_TEST_VOLATILE_EXCHANGE:    test_volatile_exchange.TestVolatileExchangeDetails,
	exchange_common.EXCHANGE_ID_TEST_FIXED_PRICE_EXCHANGE: test_fixed_price_exchange.TestFixedPriceExchangeDetails,
}
