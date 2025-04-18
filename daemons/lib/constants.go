package lib

import (
	"math"
	"math/big"
)

// BigFloatMaxUint64 returns a `big.Float` that is set to MaxUint64.
func BigFloatMaxUint64() *big.Float {
	return new(big.Float).SetUint64(math.MaxUint64)
}

// BigFloat0 returns a `big.Float` that is set to 0.
func BigFloat0() *big.Float {
	return big.NewFloat(0)
}
