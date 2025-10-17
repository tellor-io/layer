package reader

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/tellor-io/layer/daemons/utils"
)

func TestEncodeFunctionCall(t *testing.T) {
	reader := &Reader{}

	tests := []struct {
		name        string
		functionSig string
		args        []string
		wantErr     bool
	}{
		{
			name:        "No arguments function",
			functionSig: "getExchangeRate() returns (uint256)",
			args:        nil,
			wantErr:     false,
		},
		{
			name:        "Single uint256 argument",
			functionSig: "balanceOf(address) returns (uint256)",
			args:        []string{"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"},
			wantErr:     false,
		},
		{
			name:        "Multiple arguments",
			functionSig: "transfer(address,uint256) returns (bool)",
			args:        []string{"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "1000000000000000000"},
			wantErr:     false,
		},
		{
			name:        "Invalid function signature",
			functionSig: "invalidFunction",
			args:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methodID, inputData, err := reader.encodeFunctionCall(tt.functionSig, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Method ID should always be 4 bytes
			if len(methodID) != 4 {
				t.Errorf("Method ID should be 4 bytes, got %d", len(methodID))
			}

			// If args provided, input data should not be nil
			if len(tt.args) > 0 && inputData == nil {
				t.Errorf("Expected input data for arguments")
			}
		})
	}
}

func TestParseArgument(t *testing.T) {
	reader := &Reader{}

	tests := []struct {
		name      string
		arg       string
		paramType string
		wantErr   bool
	}{
		{
			name:      "uint256 value",
			arg:       "1000000000000000000",
			paramType: "uint256",
			wantErr:   false,
		},
		{
			name:      "address value",
			arg:       "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			paramType: "address",
			wantErr:   false,
		},
		{
			name:      "bool true",
			arg:       "true",
			paramType: "bool",
			wantErr:   false,
		},
		{
			name:      "bool false",
			arg:       "false",
			paramType: "bool",
			wantErr:   false,
		},
		{
			name:      "string value",
			arg:       "test string",
			paramType: "string",
			wantErr:   false,
		},
		{
			name:      "invalid uint",
			arg:       "not a number",
			paramType: "uint256",
			wantErr:   true,
		},
		{
			name:      "invalid address",
			arg:       "not an address",
			paramType: "address",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reader.parseArgument(tt.arg, tt.paramType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func TestFormatBigInt(t *testing.T) {
	tests := []struct {
		name     string
		value    *big.Int
		decimals int
		expected float64
	}{
		{
			name:     "18 decimals ETH",
			value:    big.NewInt(1000000000000000000),
			decimals: 18,
			expected: 1.0,
		},
		{
			name:     "6 decimals USDC",
			value:    big.NewInt(1000000),
			decimals: 6,
			expected: 1.0,
		},
		{
			name:     "0 decimals",
			value:    big.NewInt(100),
			decimals: 0,
			expected: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatBigInt(tt.value, tt.decimals)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// Integration test - requires real RPC endpoint
func TestReadContractIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires a real RPC endpoint
	// Set ETH_RPC_URL environment variable to run
	rpcURL := "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
	if rpcURL == "" || rpcURL == "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY" {
		t.Skip("ETH_RPC_URL not set, skipping integration test")
	}

	urls := []string{rpcURL}

	reader, err := NewReader(urls, 10)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test reading USDC decimals (should return 6)
	usdcAddress := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	result, err := reader.ReadContract(ctx, usdcAddress, "decimals() returns (uint8)", nil)
	if err != nil {
		t.Fatalf("Failed to read contract: %v", err)
	}
	value := new(big.Int).SetBytes(result)
	if value.Cmp(big.NewInt(6)) != 0 {
		t.Errorf("Expected USDC decimals to be 6, got %s", value.String())
	}
}
