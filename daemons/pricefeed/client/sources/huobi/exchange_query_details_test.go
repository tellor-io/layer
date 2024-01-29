package huobi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/huobi"
)

func TestHuobiUrl(t *testing.T) {
	require.Equal(t, "https://api.huobi.pro/market/tickers", huobi.HuobiDetails.Url)
}

func TestHuobiIsMultiMarket(t *testing.T) {
	require.True(t, huobi.HuobiDetails.IsMultiMarket)
}
