package types

import (
	"cosmossdk.io/collections"
)

type CustomPairRange[K1, K2 any] struct {
	start *collections.RangeKey[collections.Pair[K1, K2]]
	end   *collections.RangeKey[collections.Pair[K1, K2]]
	order collections.Order

	err error
}

func (p *CustomPairRange[K1, K2]) RangeValues() (start, end *collections.RangeKey[collections.Pair[K1, K2]], order collections.Order, err error) {
	if p.err != nil {
		return nil, nil, 0, err
	}
	return p.start, p.end, p.order, nil
}

func NewPrefixInBetween[K1, K2 any](start, end K1) *CustomPairRange[K1, K2] {
	return &CustomPairRange[K1, K2]{
		start: collections.RangeKeyExact(collections.PairPrefix[K1, K2](start)),
		end:   collections.RangeKeyPrefixEnd(collections.PairPrefix[K1, K2](end)),
	}
}

func (p *CustomPairRange[K1, K2]) Descending() *CustomPairRange[K1, K2] {
	p.order = collections.OrderDescending
	return p
}
