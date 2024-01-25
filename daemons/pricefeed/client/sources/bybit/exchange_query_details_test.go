package bybit_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/bybit"
)

func TestBybitUrl(t *testing.T) {
	require.Equal(t, "https://api.bybit.com/v5/market/tickers?category=spot", bybit.BybitDetails.Url)
}

func TestBybitIsMultiMarket(t *testing.T) {
	require.True(t, bybit.BybitDetails.IsMultiMarket)
}
