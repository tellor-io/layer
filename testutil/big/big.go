// Package big provides testing utility methods for the "math/big" library.
package big

import (
	"fmt"
	"math/big"
)

var bigPow10Memo = warmCache()

// MustFirst is used for returning the first value of the SetString
// method on a `*big.Int` or `*big.Rat`. This will panic if the conversion fails.
func MustFirst[T *big.Int | *big.Rat](n T, success bool) T {
	if !success {
		panic("Conversion failed")
	}
	return n
}

// Int64MulPow10 returns the result of `val * 10^exponent`, in *big.Int.
func Int64MulPow10(
	val int64,
	exponent uint64,
) (
	result *big.Int,
) {
	return new(big.Int).Mul(
		big.NewInt(val),
		BigPow10(exponent),
	)
}

// BigPow10 returns the result of `10^exponent`. Caches all calculated values and
// re-uses cached values in any following calls to BigPow10.
func BigPow10(exponent uint64) *big.Int {
	result := bigPow10Helper(exponent)
	// Copy the result, such that no values can be modified by reference in the
	// `bigPow10Memo` cache.
	copy := new(big.Int).Set(result)
	return copy
}

func bigPow10Helper(exponent uint64) *big.Int {
	m, ok := bigPow10Memo[exponent]
	if ok {
		return m
	}

	// Subdivide the exponent and recursively calculate each result, then multiply
	// both results together (given that `10^exponent = 10^(exponent / 2) *
	// 10^(exponent - (exponent / 2))`.
	e1 := exponent / 2
	e2 := exponent - e1
	return new(big.Int).Mul(bigPow10Helper(e1), bigPow10Helper(e2))
}

// warmCache is used to populate `bigPow10Memo` with the most common exponent requests. Since,
// none of the exponents should ever be invalid - panic immediately if an exponent is cannot be
// parsed.
func warmCache() map[uint64]*big.Int {
	exponentString := "1"
	bigExponentValues := make(map[uint64]*big.Int, 100)
	for i := 0; i < 100; i++ {
		bigValue, ok := new(big.Int).SetString(exponentString, 0)

		if !ok {
			panic(fmt.Sprintf("Failed to get big from string for exponent memo: %v", exponentString))
		}

		bigExponentValues[uint64(i)] = bigValue
		exponentString += "0"
	}

	return bigExponentValues
}
