package okx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/okx"
)

func TestOkxUrl(t *testing.T) {
	require.Equal(t, "https://www.okx.com/api/v5/market/tickers?instType=SPOT", okx.OkxDetails.Url)
}

func TestOkxIsMultiMarket(t *testing.T) {
	require.True(t, okx.OkxDetails.IsMultiMarket)
}
