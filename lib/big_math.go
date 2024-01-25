package lib

import (
	"fmt"
	"math/big"
)

// bigPow10Memo is a cache of the most common exponent value requests. Since bigPow10Memo will be
// accessed from different go-routines, the map should only ever be read from or collision
// could occur.
var bigPow10Memo = warmCache()

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
		exponentString = exponentString + "0"
	}

	return bigExponentValues
}
