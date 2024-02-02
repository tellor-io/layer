package gate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/gate"
)

func TestGateUrl(t *testing.T) {
	require.Equal(t, "https://api.gateio.ws/api/v4/spot/tickers", gate.GateDetails.Url)
}

func TestGateIsMultiMarket(t *testing.T) {
	require.True(t, gate.GateDetails.IsMultiMarket)
}
