package metrics_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/exchange_common"
	"github.com/tellor-io/layer/daemons/lib/metrics"
	pricefeedmetrics "github.com/tellor-io/layer/daemons/pricefeed/metrics"
	"github.com/tellor-io/layer/daemons/testutil/daemons/pricefeed/exchange_config"
)

const (
	INVALID_ID = 10000000
)

func TestGetLabelForMarketIdSuccess(t *testing.T) {
	pricefeedmetrics.SetMarketPairForTelemetry(exchange_config.MARKET_BTC_USD, "BTCUSD")
	require.Equal(
		t,
		metrics.GetLabelForStringValue(metrics.MarketId, "BTCUSD"),
		pricefeedmetrics.GetLabelForMarketId(exchange_config.MARKET_BTC_USD),
	)
}

func TestGetLabelForMarketIdFailure(t *testing.T) {
	require.Equal(
		t,
		metrics.GetLabelForStringValue(metrics.MarketId, fmt.Sprintf("invalid_id:%d", INVALID_ID)),
		pricefeedmetrics.GetLabelForMarketId(INVALID_ID),
	)
}

func TestGetLabelForExchangeId(t *testing.T) {
	require.Equal(
		t,
		metrics.GetLabelForStringValue(metrics.ExchangeId, exchange_common.EXCHANGE_ID_BINANCE_US),
		pricefeedmetrics.GetLabelForExchangeId(exchange_common.EXCHANGE_ID_BINANCE_US),
	)
}
