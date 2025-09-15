package contract_handlers

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
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

func TestRocketPoolETHHandler_FetchValue(t *testing.T) {
	tests := []struct {
		name            string
		contractResult  string // Hex encoded result
		ethPrice        uint64
		expectedValue   float64
		expectError     bool
		errorMessage    string
		skipMarketParam bool
	}{
		{
			name:           "successful fetch",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1080000000000000000).Bytes(), 32)), // 1.08 ETH per rETH
			ethPrice:       4500000000,                                                                       // $4500 with 6 decimals
			expectedValue:  4860000000,                                                                       // 1.08 * 4500 * 1e6
			expectError:    false,
		},
		{
			name:           "higher exchange rate",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1150000000000000000).Bytes(), 32)), // 1.15 ETH per rETH
			ethPrice:       3800000000,                                                                       // $3800 with 6 decimals
			expectedValue:  4370000000,                                                                       // 1.15 * 3800 * 1e6
			expectError:    false,
		},
		{
			name:            "no ETH market param",
			contractResult:  "0x" + hex.EncodeToString(padLeft(big.NewInt(1080000000000000000).Bytes(), 32)),
			skipMarketParam: true,
			expectError:     true,
			errorMessage:    "no valid ETH-USD",
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
				// Ensure ETH market param exists
				originalParam := constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID]
				constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID] = &marketParam.MarketParam{
					Id:       exchange_common.ETHUSD_ID,
					Exponent: -6,
				}
				defer func() {
					if originalParam != nil {
						constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID] = originalParam
					} else {
						delete(constants.StaticMarketParamsConfig, exchange_common.ETHUSD_ID)
					}
				}()

				// Mock price response
				now := time.Now()
				priceCache.UpdatePrices([]*pricefeedservertypes.MarketPriceUpdate{
					{
						MarketId: exchange_common.ETHUSD_ID,
						ExchangePrices: []*pricefeedservertypes.ExchangePrice{
							{
								ExchangeId:     "test",
								Price:          tt.ethPrice,
								LastUpdateTime: &now,
							},
						},
					},
				})
			} else {
				// Remove market param to test error case
				originalParam := constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID]
				delete(constants.StaticMarketParamsConfig, exchange_common.ETHUSD_ID)
				defer func() {
					if originalParam != nil {
						constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID] = originalParam
					}
				}()
			}

			// Create handler and test
			handler := &RocketPoolETHHandler{}
			ctx := context.Background()

			value, err := handler.FetchValue(ctx, contractReader, priceCache)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestRocketPoolETHHandler_CalculationPrecision(t *testing.T) {
	tests := []struct {
		name             string
		exchangeRate     *big.Int
		ethPrice         uint64
		expectedRawValue *big.Int
	}{
		{
			name:             "standard calculation",
			exchangeRate:     big.NewInt(1080000000000000000), // 1.08
			ethPrice:         4500000000,                      // $4500
			expectedRawValue: big.NewInt(4860000000),          // 4860 with 6 decimals
		},
		{
			name:             "small exchange rate",
			exchangeRate:     big.NewInt(1010000000000000000), // 1.01
			ethPrice:         5000000000,                      // $5000
			expectedRawValue: big.NewInt(5050000000),          // 5050 with 6 decimals
		},
		{
			name:             "large exchange rate",
			exchangeRate:     big.NewInt(1250000000000000000), // 1.25
			ethPrice:         4000000000,                      // $4000
			expectedRawValue: big.NewInt(5000000000),          // 5000 with 6 decimals
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the calculation from the handler
			ethPriceBig := new(big.Int).SetUint64(tt.ethPrice)
			value := new(big.Int).Mul(tt.exchangeRate, ethPriceBig)
			divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
			value.Div(value, divisor)

			assert.Equal(t, tt.expectedRawValue.String(), value.String())
		})
	}
}

// Integration test - tests the actual contract read functionality
func TestRocketPoolETHHandler_ContractRead_Integration(t *testing.T) {
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

	// Test the actual contract call to get rETH exchange rate
	result, err := contractReader.ReadContract(ctx, RETH_CONTRACT, "getExchangeRate() returns (uint256)", nil)
	require.NoError(t, err, "Failed to read rETH contract")

	// Parse the result
	rethExchangeRate := new(big.Int).SetBytes(result)
	
	// Convert to float for validation
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	rateFloat := new(big.Float).SetInt(rethExchangeRate)
	rate := new(big.Float).Quo(rateFloat, divisor)
	rateValue, _ := rate.Float64()

	// rETH should always be worth more than 1 ETH (typically 1.05 to 1.15)
	assert.Greater(t, rateValue, 1.0, "rETH should be worth more than 1 ETH")
	assert.Less(t, rateValue, 2.0, "rETH exchange rate seems unreasonably high")

	fmt.Printf("Integration test: 1 rETH = %.4f ETH\n", rateValue)
}