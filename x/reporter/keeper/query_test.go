package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestReportersQuery(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	for i := 0; i < 10; i++ {
		err := k.Reporters.Set(ctx, sample.AccAddressBytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb))
		require.NoError(t, err)
	}
	res, err := querier.Reporters(ctx, &types.QueryReportersRequest{})
	require.NoError(t, err)
	require.Len(t, res.Reporters, 10)
}

func TestSelectorReporterQuery(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	selector := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	err := k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1))
	require.NoError(t, err)
	res, err := querier.SelectorReporter(ctx, &types.QuerySelectorReporterRequest{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.Equal(t, reporterAddr.String(), res.Reporter)
}

func TestAllowedAmountQuery(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)

	// set the last stored tracked amount
	amt := math.NewInt(1000)
	err := k.Tracker.Set(ctx, types.StakeTracker{Amount: amt})
	require.NoError(t, err)

	sk.On("TotalBondedTokens", ctx).Return(amt, nil)
	res, err := querier.AllowedAmount(ctx, &types.QueryAllowedAmountRequest{})
	require.NoError(t, err)

	expectedAllowedAmount := math.NewInt(50)
	require.Equal(t, expectedAllowedAmount, res.StakingAmount)
	require.Equal(t, expectedAllowedAmount.Neg(), res.UnstakingAmount)
}
