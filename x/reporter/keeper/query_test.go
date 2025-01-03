package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func TestReportersQuery(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
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
	k, _, _, _, _, ctx, _ := setupKeeper(t)
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
	k, _, sk, _, _, ctx, _ := setupKeeper(t)
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

func TestNumOfSelectorsByReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)

	reporterAddr := sample.AccAddressBytes()
	for i := 0; i < 10; i++ {
		err := k.Selectors.Set(ctx, sample.AccAddressBytes(), types.NewSelection(reporterAddr, 1))
		require.NoError(t, err)
	}

	res, err := querier.NumOfSelectorsByReporter(ctx, &types.QueryNumOfSelectorsByReporterRequest{ReporterAddress: reporterAddr.String()})
	require.NoError(t, err)
	require.Equal(t, res.NumOfSelectors, int32(10))
}

func TestSpaceAvailableByReporter(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)

	reporterAddr := sample.AccAddressBytes()
	for i := 0; i < 10; i++ {
		err := k.Selectors.Set(ctx, sample.AccAddressBytes(), types.NewSelection(reporterAddr, 1))
		require.NoError(t, err)
	}

	res, err := querier.SpaceAvailableByReporter(ctx, &types.QuerySpaceAvailableByReporterRequest{ReporterAddress: reporterAddr.String()})
	require.NoError(t, err)
	require.Equal(t, res.SpaceAvailable, int32(90))
}

func TestAllowedAmountExpiration(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	ctx = ctx.WithBlockTime(time.Now())

	expiration := ctx.BlockTime().Add(1)
	require.NoError(t, k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)}))

	res, err := querier.AllowedAmountExpiration(ctx, &types.QueryAllowedAmountExpirationRequest{})
	require.NoError(t, err)
	require.Equal(t, res.Expiration, uint64(ctx.BlockTime().Add(1).UnixMilli()))
}

func TestRewardClaimStatus(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	addy := sample.AccAddressBytes()
	require.NoError(t, k.ClaimStatus.Set(ctx, collections.Join(addy.Bytes(), uint64(10)), true))

	res, err := querier.RewardClaimStatus(ctx, &types.QueryRewardClaimStatusRequest{Id: 10, SelectorAddress: addy.String()})
	require.NoError(t, err)
	require.Equal(t, res.Status, true)

	res, err = querier.RewardClaimStatus(ctx, &types.QueryRewardClaimStatusRequest{Id: 1, SelectorAddress: addy.String()})
	require.NoError(t, err)
	require.Equal(t, res.Status, false)
}
