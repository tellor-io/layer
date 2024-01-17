package coinbase_pro_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/coinbase_pro"
)

func TestCoinbaseProUrl(t *testing.T) {
	require.Equal(t, "https://api.pro.coinbase.com/products/$/ticker", coinbase_pro.CoinbaseProDetails.Url)
}

func TestCoinbaseProIsMultiMarket(t *testing.T) {
	require.False(t, coinbase_pro.CoinbaseProDetails.IsMultiMarket)
}
