package kraken_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/kraken"
)

func TestKrakenUrl(t *testing.T) {
	require.Equal(t, "https://api.kraken.com/0/public/Ticker", kraken.KrakenDetails.Url)
}

func TestKrakenIsMultiMarket(t *testing.T) {
	require.True(t, kraken.KrakenDetails.IsMultiMarket)
}
