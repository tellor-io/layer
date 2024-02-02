package crypto_com_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/pricefeed/client/sources/crypto_com"
)

func TestCryptoComUrl(t *testing.T) {
	require.Equal(t, "https://api.crypto.com/v2/public/get-ticker", crypto_com.CryptoComDetails.Url)
}

func TestCryptoComIsMultiMarket(t *testing.T) {
	require.True(t, crypto_com.CryptoComDetails.IsMultiMarket)
}
