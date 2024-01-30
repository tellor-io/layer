package kucoin_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/kucoin"
)

func TestKucoinUrl(t *testing.T) {
	require.Equal(t, "https://api.kucoin.com/api/v1/market/allTickers", kucoin.KucoinDetails.Url)
}

func TestKucoinIsMultiMarket(t *testing.T) {
	require.True(t, kucoin.KucoinDetails.IsMultiMarket)
}
