package combined_handler

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

func TestSFRXUSDHandler_Success(t *testing.T) {
	totalAssets := big.NewInt(1050000000000000000)
	totalSupply := big.NewInt(1000000000000000000)
	rpcServer := createTestRPCServer(t, totalAssets, totalSupply)
	defer rpcServer.Close()

	contractReader, err := contractreader.NewReader([]string{rpcServer.URL}, 10)
	require.NoError(t, err)
	defer contractReader.Close()

	// Create RPC readers for APIs
	coingeckoServer := createCoinGeckoServer(1.02)
	defer coingeckoServer.Close()

	curveServer := createCurveServer(1.01)
	defer curveServer.Close()

	coinpaprikaServer := createCoinPaprikaServer(1.03)
	defer coinpaprikaServer.Close()

	coingeckoReader, err := rpcreader.NewReader(
		coingeckoServer.URL,
		"GET",
		"",
		nil,
		[]string{"frax", "usd"},
		5000,
	)
	require.NoError(t, err)

	curveReader, err := rpcreader.NewReader(
		curveServer.URL,
		"GET",
		"",
		nil,
		[]string{"data", "usd_price"},
		5000,
	)
	require.NoError(t, err)

	coinpaprikaReader, err := rpcreader.NewReader(
		coinpaprikaServer.URL,
		"GET",
		"",
		nil,
		[]string{"quotes", "USD", "price"},
		5000,
	)
	require.NoError(t, err)

	contractReaders := map[string]*contractreader.Reader{
		"ethereum": contractReader,
	}

	rpcReaders := map[string]*rpcreader.Reader{
		"coingecko":   coingeckoReader,
		"curve":       curveReader,
		"coinpaprika": coinpaprikaReader,
	}

	priceCache := pricefeedservertypes.NewMarketToExchangePrices(1 * time.Minute)

	handler := &SFRXUSDPriceHandler{}
	ctx := context.Background()

	price, err := handler.FetchValue(ctx, contractReaders, rpcReaders, priceCache, 2, 50.0)

	require.NoError(t, err)
	expectedPrice := (float64(totalAssets.Int64()) / float64(totalSupply.Int64())) * 1.02
	require.Equal(t, expectedPrice, price)
}

func TestSFRXUSDHandler_MissingContractReader(t *testing.T) {
	priceCache := pricefeedservertypes.NewMarketToExchangePrices(1 * time.Minute)
	handler := &SFRXUSDPriceHandler{}
	ctx := context.Background()

	_, err := handler.FetchValue(ctx, map[string]*contractreader.Reader{}, make(map[string]*rpcreader.Reader), priceCache, 2, 50.0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ethereum contract reader not found")
}

func TestSFRXUSDHandler_ZeroTotalSupply(t *testing.T) {
	totalAssets := big.NewInt(1000)
	totalSupply := big.NewInt(0)
	rpcServer := createTestRPCServer(t, totalAssets, totalSupply)
	defer rpcServer.Close()

	contractReader, err := contractreader.NewReader([]string{rpcServer.URL}, 10)
	require.NoError(t, err)
	defer contractReader.Close()

	priceCache := pricefeedservertypes.NewMarketToExchangePrices(1 * time.Minute)
	handler := &SFRXUSDPriceHandler{}
	ctx := context.Background()

	_, err = handler.FetchValue(ctx, map[string]*contractreader.Reader{"ethereum": contractReader}, make(map[string]*rpcreader.Reader), priceCache, 2, 50.0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid total supply: zero")
}

func TestSFRXUSDHandler_InsufficientSources(t *testing.T) {
	totalAssets := big.NewInt(1000)
	totalSupply := big.NewInt(1000)
	rpcServer := createTestRPCServer(t, totalAssets, totalSupply)
	defer rpcServer.Close()

	contractReader, err := contractreader.NewReader([]string{rpcServer.URL}, 10)
	require.NoError(t, err)
	defer contractReader.Close()

	// Only one source
	coingeckoServer := createCoinGeckoServer(1.02)
	defer coingeckoServer.Close()

	coingeckoReader, err := rpcreader.NewReader(
		coingeckoServer.URL,
		"GET",
		"",
		nil,
		[]string{"frax", "usd"},
		5000,
	)
	require.NoError(t, err)

	priceCache := pricefeedservertypes.NewMarketToExchangePrices(1 * time.Minute)
	handler := &SFRXUSDPriceHandler{}
	ctx := context.Background()

	_, err = handler.FetchValue(ctx, map[string]*contractreader.Reader{"ethereum": contractReader}, map[string]*rpcreader.Reader{"coingecko": coingeckoReader}, priceCache, 2, 50.0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient FRX/USD prices")
}

// Helpers
func createTestRPCServer(t *testing.T, totalAssets, totalSupply *big.Int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "eth_call" {
			methodID := ""
			if params, ok := req["params"].([]interface{}); ok && len(params) > 0 {
				if callObj, ok := params[0].(map[string]interface{}); ok {
					if input, ok := callObj["input"].(string); ok && len(input) >= 10 {
						methodID = input[:10]
					}
				}
			}

			var result string
			// totalAssets() method ID: 0x01e1d114
			// totalSupply() method ID: 0x18160ddd
			switch methodID {
			case "0x01e1d114": // totalAssets()
				result = encodeBigInt(totalAssets)
			case "0x18160ddd": // totalSupply()
				result = encodeBigInt(totalSupply)
			default:
				result = "0x"
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  result,
			})
		}
	}))
}

func createCoinGeckoServer(price float64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if price > 0 {
			json.NewEncoder(w).Encode(map[string]map[string]float64{
				"frax": {"usd": price},
			})
		}
	}))
}

func createCurveServer(price float64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if price > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]float64{"usd_price": price},
			})
		}
	}))
}

func createCoinPaprikaServer(price float64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if price > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"quotes": map[string]interface{}{
					"USD": map[string]float64{"price": price},
				},
			})
		}
	}))
}

func encodeBigInt(value *big.Int) string {
	bytes := value.Bytes()
	if len(bytes) > 32 {
		bytes = bytes[len(bytes)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(bytes):], bytes)
	return "0x" + hexEncode(padded)
}

func hexEncode(data []byte) string {
	const hextable = "0123456789abcdef"
	output := make([]byte, len(data)*2)
	for i, b := range data {
		output[i*2] = hextable[b>>4]
		output[i*2+1] = hextable[b&0x0f]
	}
	return string(output)
}
