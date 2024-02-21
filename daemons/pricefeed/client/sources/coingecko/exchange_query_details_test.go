package coingecko_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/coingecko"
)

func TestCoingeckoUrl(t *testing.T) {
	require.Equal(t, "https://api.coingecko.com/api/v3/simple/price?ids=$&vs_currencies=usd", coingecko.CoingeckoDetails.Url)
}

func TestCoingeckoIsMultiMarket(t *testing.T) {
	require.True(t, coingecko.CoingeckoDetails.IsMultiMarket)
}
