package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestMergeFeeOriginsOrder(t *testing.T) {
	origin := func(delegator, validator string, amount int64) *types.TokenOriginInfo {
		return &types.TokenOriginInfo{DelegatorAddress: []byte(delegator), ValidatorAddress: []byte(validator), Amount: math.NewInt(amount)}
	}
	// Historical duplicates contribute to the first slot for their pair.
	payment := []*types.TokenOriginInfo{origin("b", "x", 2), origin("a", "y", 4), origin("a", "x", 1)}
	previous := []*types.TokenOriginInfo{origin("a", "x", 10), origin("c", "x", 30), origin("a", "y", 3), origin("b", "x", 20), origin("b", "x", 5), origin("c", "y", 6)}
	merged := mergeFeeOrigins(payment, previous)
	require.Equal(t, []*types.TokenOriginInfo{
		origin("b", "x", 27), origin("a", "y", 7), origin("a", "x", 11),
		origin("c", "x", 30), origin("c", "y", 6),
	}, merged)

	// A new pair occupies its payment slot ahead of an untouched old pair.
	merged = mergeFeeOrigins([]*types.TokenOriginInfo{origin("d", "x", 1)}, []*types.TokenOriginInfo{origin("a", "x", 2)})
	require.Equal(t, []*types.TokenOriginInfo{origin("d", "x", 1), origin("a", "x", 2)}, merged)
}
