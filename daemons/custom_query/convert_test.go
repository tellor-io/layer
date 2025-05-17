package customquery_test

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
)

func TestConversion(t *testing.T) {
	testCases := []struct {
		value    float64
		abiType  string
		name     string
		decimals int
		bitSize  int
	}{
		// Unsigned integers
		{123, "uint8", "uint8", 1, 8},
		{1234, "uint16", "uint16", 1, 16},
		{123456, "uint32", "uint32", 1, 32},
		{9876543210, "uint64", "uint64", 1, 64},
		{123456789012345, "uint256", "uint256", 1, 256},

		// Signed integers
		{123, "int8", "int8", 1, 8},
		{-123, "int8", "negative int8", 1, 8},
		{1234, "int16", "int16", 1, 16},
		{-1234, "int16", "negative int16", 1, 16},
		{2147483647, "int32", "max int32", 1, 32},
		{-2147483648, "int32", "min int32", 1, 32},
		{-9876543210, "int64", "negative int64", 1, 64},

		// Unsigned fixed
		{123.45, "ufixed16x2", "ufixed16x2", 2, 16},
		{0.001234, "ufixed32x6", "ufixed32x6", 6, 32},
		{123.456789, "ufixed64x6", "ufixed64x6", 6, 64},
		{1.1578305026317555, "ufixed256x18", "ufixed256x18 test case", 18, 256},

		// Signed fixed
		{123.45, "fixed16x2", "fixed16x2", 2, 16},
		{-123.45, "fixed16x2", "negative fixed16x2", 2, 16},
		{0.001234, "fixed32x6", "fixed32x6", 6, 32},
		{-0.001234, "fixed32x6", "negative fixed32x6", 6, 32},
		{-123.456789, "fixed64x6", "negative fixed64x6", 6, 64},
		{-1.1578305026317555, "fixed256x18", "fixed256x18 test case", 18, 256},
	}

	// Run tests
	for _, tc := range testCases {
		result, err := customquery.AbiNumberEncoder(tc.value, tc.abiType)
		require.NotEqual(t, "", result)
		require.NoError(t, err, "Error encoding value: %v", err)
		v, err := DecodeHexString(result, tc.abiType, tc.bitSize, tc.decimals)
		require.NoError(t, err, "Error decoding value: %v", err)
		require.Equal(t, tc.value, v, "Expected value: %v, got: %v", tc.value, v)
	}
}

func DecodeHexString(hexStr, abiType string, bitSize, decimals int) (float64, error) {
	bigInt, success := new(big.Int).SetString(hexStr, 16)
	if !success {
		return 0, fmt.Errorf("failed to parse hex value: %s", hexStr)
	}

	switch {
	case strings.HasPrefix(abiType, "uint"):
		result, _ := new(big.Float).SetInt(bigInt).Float64()
		return result, nil

	case strings.HasPrefix(abiType, "int"):
		if len(hexStr) > 0 && hexStr[0] >= '8' {
			twoToBitSize := new(big.Int).Lsh(big.NewInt(1), uint(bitSize))
			bigInt.Sub(twoToBitSize, bigInt)
			bigInt.Neg(bigInt)
		}

		result, _ := new(big.Float).SetInt(bigInt).Float64()
		return result, nil

	case strings.HasPrefix(abiType, "ufixed"):
		result, _ := new(big.Float).SetInt(bigInt).Float64()
		return result / math.Pow10(decimals), nil

	case strings.HasPrefix(abiType, "fixed"):
		if len(hexStr) > 0 && hexStr[0] >= '8' {
			twoToBitSize := new(big.Int).Lsh(big.NewInt(1), uint(bitSize))
			bigInt.Sub(twoToBitSize, bigInt)
			bigInt.Neg(bigInt)
		}

		result, _ := new(big.Float).SetInt(bigInt).Float64()
		return result / math.Pow10(decimals), nil

	default:
		return 0, fmt.Errorf("unsupported ABI type: %s", abiType)
	}
}
