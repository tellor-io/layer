package contract_handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	"github.com/tellor-io/layer/daemons/exchange_common"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/daemons"
	pricefeedtypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

// Helper function to create a test RPC server
func createTestRPCServer(t *testing.T, expectedResult string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Check if it's an eth_call
		if req["method"] == "eth_call" {
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result":  expectedResult,
			}
			err = json.NewEncoder(w).Encode(response)
			require.NoError(t, err)
		}
	}))
}

func TestWSTETHHandler_FetchValue(t *testing.T) {
	tests := []struct {
		name                string
		contractResult      string // Hex encoded result
		stEthPrice          uint64
		marketParamExponent int32
		expectedValue       float64
		expectError         bool
		errorMessage        string
		skipMarketParam     bool
	}{
		{
			name:                "successful fetch with exponent -6",
			contractResult:      "0x" + hex.EncodeToString(padLeft(big.NewInt(1213540318890078223).Bytes(), 32)), // ~1.2135 stETH per wstETH
			stEthPrice:          4700000000,                                                                      // $4700 with 6 decimals
			marketParamExponent: -6,
			expectedValue:       5703.639497, // 1.2135 * 4700
			expectError:         false,
		},
		{
			name:                "successful fetch with exponent -8",
			contractResult:      "0x" + hex.EncodeToString(padLeft(big.NewInt(1213540318890078223).Bytes(), 32)),
			stEthPrice:          470000000000, // $4700 with 8 decimals
			marketParamExponent: -8,
			expectedValue:       5703.639497,
			expectError:         false,
		},
		{
			name:            "no stETH market param",
			contractResult:  "0x" + hex.EncodeToString(padLeft(big.NewInt(1213540318890078223).Bytes(), 32)),
			skipMarketParam: true,
			expectError:     true,
			errorMessage:    "no valid stETH-USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test RPC server
			server := createTestRPCServer(t, tt.contractResult)
			defer server.Close()

			// Create reader with test server URL
			contractReader, err := reader.NewReader([]string{server.URL}, 10)
			require.NoError(t, err)
			defer contractReader.Close()

			// Setup price cache
			priceCache := pricefeedtypes.NewMarketToExchangePrices(1 * time.Minute)

			// Configure market param if needed
			if !tt.skipMarketParam {
				// Temporarily override the market param for testing
				originalParam := constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID]
				constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID] = &marketParam.MarketParam{
					Id:       exchange_common.STETHUSD_ID,
					Exponent: tt.marketParamExponent,
				}
				defer func() {
					if originalParam != nil {
						constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID] = originalParam
					} else {
						delete(constants.StaticMarketParamsConfig, exchange_common.STETHUSD_ID)
					}
				}()

				// Mock price response
				now := time.Now()
				priceCache.UpdatePrices([]*pricefeedservertypes.MarketPriceUpdate{
					{
						MarketId: exchange_common.STETHUSD_ID,
						ExchangePrices: []*pricefeedservertypes.ExchangePrice{
							{
								ExchangeId:     "test",
								Price:          tt.stEthPrice,
								LastUpdateTime: &now,
							},
						},
					},
				})
			} else {
				// Remove market param to test error case
				originalParam := constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID]
				delete(constants.StaticMarketParamsConfig, exchange_common.STETHUSD_ID)
				defer func() {
					if originalParam != nil {
						constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID] = originalParam
					}
				}()
			}

			// Create handler and test
			handler := &WSTETHHandler{}
			ctx := context.Background()

			value, err := handler.FetchValue(ctx, contractReader, priceCache)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				// Allow small floating point differences
				assert.InDelta(t, tt.expectedValue, value, 0.01)
			}
		})
	}
}

func TestWSTETHHandler_ExponentCalculation(t *testing.T) {
	tests := []struct {
		name     string
		exponent int32
		value    uint64
		expected float64
	}{
		{
			name:     "negative exponent -6",
			exponent: -6,
			value:    5703760852, // Value after dividing by 10^18
			expected: 5703.760852,
		},
		{
			name:     "negative exponent -8",
			exponent: -8,
			value:    570376085200, // Same USD value but with 8 decimals
			expected: 5703.760852,
		},
		{
			name:     "negative exponent -4",
			exponent: -4,
			value:    57037608, // Same USD value but with 4 decimals
			expected: 5703.7608,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate using the same logic as in the handler
			scaleFactor := math.Pow(10, float64(tt.exponent))
			result := float64(tt.value) * scaleFactor

			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

// Helper function to pad bytes to specific length
func padLeft(data []byte, size int) []byte {
	if len(data) >= size {
		return data
	}
	padded := make([]byte, size)
	copy(padded[size-len(data):], data)
	return padded
}

// Integration test - tests the actual contract read functionality
func TestWSTETHHandler_ContractRead_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use public RPC endpoints that don't require authentication
	rpcURLs := []string{
		"https://eth.llamarpc.com",
		"https://ethereum.publicnode.com",
		"https://rpc.ankr.com/eth",
	}

	contractReader, err := reader.NewReader(rpcURLs, 30)
	if err != nil {
		t.Skipf("Failed to create reader: %v", err)
	}
	defer contractReader.Close()

	ctx := context.Background()

	// Test the actual contract call to get stETH per wstETH
	result, err := contractReader.ReadContract(ctx, WSTETH_CONTRACT, "getStETHByWstETH(uint256) returns (uint256)", []string{"1000000000000000000"})
	require.NoError(t, err, "Failed to read wstETH contract")

	// Parse the result
	stEthPerWstEth := new(big.Int).SetBytes(result)

	// Convert to float for validation
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	ratioFloat := new(big.Float).SetInt(stEthPerWstEth)
	ratio := new(big.Float).Quo(ratioFloat, divisor)
	ratioValue, _ := ratio.Float64()

	// wstETH should always be worth more than 1 stETH (typically 1.1 to 1.2)
	assert.Greater(t, ratioValue, 1.0, "wstETH should be worth more than 1 stETH")
	assert.Less(t, ratioValue, 2.0, "wstETH ratio seems unreasonably high")

	fmt.Printf("Integration test: 1 wstETH = %.4f stETH\n", ratioValue)
}
