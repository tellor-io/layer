package bitstamp_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/bitstamp"
)

func TestBitstampUrl(t *testing.T) {
	require.Equal(t, "https://www.bitstamp.net/api/v2/ticker/", bitstamp.BitstampDetails.Url)
}

func TestBitstampIsMultiMarket(t *testing.T) {
	require.True(t, bitstamp.BitstampDetails.IsMultiMarket)
}
