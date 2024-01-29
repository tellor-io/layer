package sources_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources"
)

func TestExchangeError(t *testing.T) {
	error := sources.NewExchangeError("exchange", "error")
	var exchangeError sources.ExchangeError
	found := errors.As(error, &exchangeError)
	require.True(t, found)
	require.Equal(t, error, exchangeError)
	require.Equal(t, "exchange", exchangeError.GetExchangeId())
}
