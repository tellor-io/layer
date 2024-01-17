package bitfinex_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/bitfinex"
)

func TestBitfinexUrl(t *testing.T) {
	require.Equal(t, "https://api-pub.bitfinex.com/v2/tickers?symbols=ALL", bitfinex.BitfinexDetails.Url)
}

func TestBitfinexIsMultiMarket(t *testing.T) {
	require.True(t, bitfinex.BitfinexDetails.IsMultiMarket)
}
