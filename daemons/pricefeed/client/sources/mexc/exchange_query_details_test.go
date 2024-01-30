package mexc_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/mexc"
)

func TestMexcUrl(t *testing.T) {
	require.Equal(t, "https://www.mexc.com/open/api/v2/market/ticker", mexc.MexcDetails.Url)
}

func TestMexcIsMultiMarket(t *testing.T) {
	require.True(t, mexc.MexcDetails.IsMultiMarket)
}
