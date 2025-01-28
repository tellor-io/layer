package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestReportersQuery(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	for i := 0; i < 10; i++ {
		err := k.Reporters.Set(ctx, sample.AccAddressBytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya))
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
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
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

func TestAvailableTips(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	require := require.New(t)

	selectorAddr := sample.AccAddressBytes()

	cleanup := func() {
		iter, err := k.SelectorTips.Iterate(ctx, nil)
		require.NoError(err)
		defer iter.Close()

		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(err)
			require.NoError(k.SelectorTips.Remove(ctx, key))
		}
	}

	testCases := []struct {
		name     string
		setup    func()
		req      *types.QueryAvailableTipsRequest
		err      bool
		expected math.LegacyDec
	}{
		{
			name: "nil request",
			req:  nil,
			err:  true,
		},
		{
			name:     "no tips",
			req:      &types.QueryAvailableTipsRequest{SelectorAddress: selectorAddr.String()},
			err:      true,
			expected: math.LegacyZeroDec(),
		},
		{
			name: "one tip",
			setup: func() {
				err := k.SelectorTips.Set(ctx, selectorAddr, math.LegacyNewDec(100*1e6))
				require.NoError(err)
			},
			req:      &types.QueryAvailableTipsRequest{SelectorAddress: selectorAddr.String()},
			err:      false,
			expected: math.LegacyNewDec(100 * 1e6),
		},
		{
			name: "amount changes",
			setup: func() {
				err := k.SelectorTips.Set(ctx, selectorAddr, math.LegacyNewDec(100*1e6))
				require.NoError(err)
				err = k.SelectorTips.Set(ctx, selectorAddr, math.LegacyNewDec(200*1e6))
				require.NoError(err)
			},
			req:      &types.QueryAvailableTipsRequest{SelectorAddress: selectorAddr.String()},
			err:      false,
			expected: math.LegacyNewDec(200 * 1e6),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup()

			if tc.setup != nil {
				tc.setup()
			}
			res, err := querier.AvailableTips(ctx, tc.req)
			if tc.err {
				require.Error(err)
			} else {
				require.NoError(err)
				require.Equal(tc.expected, res.AvailableTips)
			}
		})
	}
}
