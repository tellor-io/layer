package binance_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/binance"
)

func TestBinanceUrl(t *testing.T) {
	require.Equal(t, "https://data-api.binance.vision/api/v3/ticker/24hr", binance.BinanceDetails.Url)
}

func TestBinanceUsUrl(t *testing.T) {
	require.Equal(t, "https://api.binance.us/api/v3/ticker/24hr", binance.BinanceUSDetails.Url)
}

func TestBinanceIsMultiMarket(t *testing.T) {
	require.True(t, binance.BinanceDetails.IsMultiMarket)
}

func TestBinanceUSIsMultiMarket(t *testing.T) {
	require.True(t, binance.BinanceUSDetails.IsMultiMarket)
}
