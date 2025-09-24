package contract_handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
)

func TestSUSDSHandler_FetchValue(t *testing.T) {
	tests := []struct {
		name           string
		contractResult string // Hex encoded result
		expectedValue  float64
		expectError    bool
	}{
		{
			name:           "successful fetch - 1.05 USD per sUSDS",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1050000000000000000).Bytes(), 32)), // 1.05 USDS per sUSDS
			expectedValue:  1.05,
			expectError:    false,
		},
		{
			name:           "successful fetch - 1.10 USD per sUSDS",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1100000000000000000).Bytes(), 32)), // 1.10 USDS per sUSDS
			expectedValue:  1.10,
			expectError:    false,
		},
		{
			name:           "successful fetch - exactly 1 USD per sUSDS",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1000000000000000000).Bytes(), 32)), // 1.00 USDS per sUSDS
			expectedValue:  1.00,
			expectError:    false,
		},
		{
			name:           "successful fetch - 1.025 USD per sUSDS",
			contractResult: "0x" + hex.EncodeToString(padLeft(big.NewInt(1025000000000000000).Bytes(), 32)), // 1.025 USDS per sUSDS
			expectedValue:  1.025,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test RPC server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				// Check if it's an eth_call
				if req["method"] == "eth_call" {
					response := map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      req["id"],
						"result":  tt.contractResult,
					}
					err = json.NewEncoder(w).Encode(response)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			// Create reader with test server URL
			contractReader, err := reader.NewReader([]string{server.URL}, 10)
			require.NoError(t, err)
			defer contractReader.Close()

			// Create handler and test
			handler := &SUSDSHandler{}
			ctx := context.Background()

			value, err := handler.FetchValue(ctx, contractReader, nil) // sUSDS doesn't use price cache

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expectedValue, value, 0.0001)
			}
		})
	}
}

func TestSUSDSHandler_CalculationPrecision(t *testing.T) {
	tests := []struct {
		name           string
		convertedValue *big.Int
		expectedUSD    float64
	}{
		{
			name:           "1.05 USDS",
			convertedValue: big.NewInt(1050000000000000000), // 1.05 in wei
			expectedUSD:    1.05,
		},
		{
			name:           "1.123456789 USDS",
			convertedValue: big.NewInt(1123456789000000000), // 1.123456789 in wei
			expectedUSD:    1.123456789,
		},
		{
			name:           "0.95 USDS",
			convertedValue: big.NewInt(950000000000000000), // 0.95 in wei
			expectedUSD:    0.95,
		},
		{
			name:           "2.0 USDS",
			convertedValue: big.NewInt(2000000000000000000), // 2.0 in wei
			expectedUSD:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replicate the calculation from the handler
			divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
			divisorFloat := new(big.Float).SetInt(divisor)
			valueInUsdFloat := new(big.Float).SetInt(tt.convertedValue)
			usdValue := new(big.Float).Quo(valueInUsdFloat, divisorFloat)
			value, _ := usdValue.Float64()

			assert.InDelta(t, tt.expectedUSD, value, 0.000000001)
		})
	}
}

// Integration test - tests the actual contract read functionality
func TestSUSDSHandler_ContractRead_Integration(t *testing.T) {
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

	// Test the actual contract call to convert sUSDS to USDS
	result, err := contractReader.ReadContract(ctx, SUSDS_CONTRACT, "convertToAssets(uint256) returns (uint256)", []string{"1000000000000000000"})
	require.NoError(t, err, "Failed to read sUSDS contract")

	// Parse the result
	usdsAmount := new(big.Int).SetBytes(result)
	
	// Convert to float for validation
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountFloat := new(big.Float).SetInt(usdsAmount)
	usdValue := new(big.Float).Quo(amountFloat, divisor)
	value, _ := usdValue.Float64()

	// sUSDS should be worth around $1.00-$1.20 (it's a yield-bearing stablecoin)
	assert.Greater(t, value, 0.95, "sUSDS value seems too low")
	assert.Less(t, value, 2.0, "sUSDS value seems unreasonably high")

	fmt.Printf("Integration test: 1 sUSDS = $%.6f USDS\n", value)
}