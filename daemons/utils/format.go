package utils

import (
	"math"
	"math/big"
)

// FormatBigInt converts a big.Int with decimals to a float64
func FormatBigInt(value *big.Int, decimals int) float64 {
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result := new(big.Float).SetInt(value)
	result.Quo(result, divisor)

	f, _ := result.Float64()
	return math.Round(f*1e6) / 1e6 // Round to 6 decimal places
}
