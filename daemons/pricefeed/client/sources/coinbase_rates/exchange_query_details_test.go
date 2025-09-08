package coinbase_rates_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/coinbase_rates"
)

func TestCoinbaseRatesUrl(t *testing.T) {
	require.Equal(t, "https://api.coinbase.com/v2/exchange-rates", coinbase_rates.CoinbaseRatesDetails.Url)
}

func TestCoinbaseRatesIsMultiMarket(t *testing.T) {
	require.False(t, coinbase_rates.CoinbaseRatesDetails.IsMultiMarket)
}
