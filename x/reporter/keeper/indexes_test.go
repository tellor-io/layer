package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestReporterDelegatorIndex(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)

	repAddr := sample.AccAddressBytes()
	// set reporter
	reporter := types.NewReporter(math.LegacyZeroDec(), math.ZeroInt())
	err := k.Reporters.Set(ctx, repAddr, reporter)
	require.NoError(t, err)

	// set delegator 1
	delAddr1 := sample.AccAddressBytes()
	del1 := types.NewSelection(repAddr, 1)
	err = k.Selectors.Set(ctx, delAddr1, del1)
	require.NoError(t, err)

	// set delegator 2
	delAddr2 := sample.AccAddressBytes()
	del2 := types.NewSelection(repAddr, 1)
	err = k.Selectors.Set(ctx, delAddr2, del2)
	require.NoError(t, err)

	// set delegator 3
	delAddr3 := sample.AccAddressBytes()
	del3 := types.NewSelection(repAddr, 1)
	err = k.Selectors.Set(ctx, delAddr3, del3)
	require.NoError(t, err)

	// get delegators for a reporter
	delAddrs, err := k.Selectors.Indexes.Reporter.MatchExact(ctx, repAddr)
	require.NoError(t, err)
	keys, err := delAddrs.PrimaryKeys()
	require.NoError(t, err)
	require.Len(t, keys, 3)

	defer delAddrs.Close()
	for ; delAddrs.Valid(); delAddrs.Next() {
		key, err := delAddrs.PrimaryKey()
		require.NoError(t, err)
		del, err := k.Selectors.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, repAddr.String(), del.Reporter)
	}
}
