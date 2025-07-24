package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestReportersQuery(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	for i := 0; i < 10; i++ {
		err := k.Reporters.Set(ctx, sample.AccAddressBytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker"))
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

func TestSelectionsTo(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)

	reporterAddr := sample.AccAddressBytes()
	err := k.Reporters.Set(ctx, reporterAddr, types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinLoya, "reporter_moniker"))
	require.NoError(t, err)

	// Store selector addresses so we can mock for each one
	selectorAddrs := make([]sdk.AccAddress, 10)
	for i := range 10 {
		selectorAddrs[i] = sample.AccAddressBytes()
		err := k.Selectors.Set(ctx, selectorAddrs[i], types.NewSelection(reporterAddr, 1))
		require.NoError(t, err)

		// Capture loop variables to avoid closure issues
		selectorAddr := selectorAddrs[i]

		// Mock CheckSelectorsDelegations for each selector
		sk.On("IterateDelegatorDelegations", ctx, selectorAddr, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
			fn := args.Get(2).(func(stakingtypes.Delegation) bool)
			delegation := stakingtypes.Delegation{
				DelegatorAddress: selectorAddr.String(),
				ValidatorAddress: sdk.ValAddress(selectorAddr).String(),
				Shares:           math.LegacyNewDec(1000),
			}
			validator := stakingtypes.Validator{
				OperatorAddress: sdk.ValAddress(selectorAddr).String(),
				Status:          stakingtypes.Bonded,
				Tokens:          math.NewInt(1000 * 1e6),
				DelegatorShares: math.LegacyNewDec(1000),
			}
			sk.On("GetValidator", ctx, sdk.ValAddress(selectorAddr)).Return(validator, nil).Once()
			fn(delegation)
		}).Twice()
	}

	// Add a selector with multiple delegations (3 validators with varying amounts)
	multiDelegationSelector := sample.AccAddressBytes()
	err = k.Selectors.Set(ctx, multiDelegationSelector, types.NewSelection(reporterAddr, 3))
	require.NoError(t, err)

	// Create 3 different validators with different amounts
	val1Addr := sample.AccAddressBytes()
	val2Addr := sample.AccAddressBytes()
	val3Addr := sample.AccAddressBytes()

	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: multiDelegationSelector.String(),
			ValidatorAddress: sdk.ValAddress(val1Addr).String(),
			Shares:           math.LegacyNewDec(500), // 500M tokens
		},
		{
			DelegatorAddress: multiDelegationSelector.String(),
			ValidatorAddress: sdk.ValAddress(val2Addr).String(),
			Shares:           math.LegacyNewDec(300), // 300M tokens
		},
		{
			DelegatorAddress: multiDelegationSelector.String(),
			ValidatorAddress: sdk.ValAddress(val3Addr).String(),
			Shares:           math.LegacyNewDec(200), // 200M tokens
		},
	}

	validators := []stakingtypes.Validator{
		{
			OperatorAddress: sdk.ValAddress(val1Addr).String(),
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(500 * 1e6),
			DelegatorShares: math.LegacyNewDec(500),
		},
		{
			OperatorAddress: sdk.ValAddress(val2Addr).String(),
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(300 * 1e6),
			DelegatorShares: math.LegacyNewDec(300),
		},
		{
			OperatorAddress: sdk.ValAddress(val3Addr).String(),
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(200 * 1e6),
			DelegatorShares: math.LegacyNewDec(200),
		},
	}

	// Mock CheckSelectorsDelegations for the multi-delegation selector
	sk.On("IterateDelegatorDelegations", ctx, multiDelegationSelector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for i, delegation := range delegations {
			valAddr, _ := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			sk.On("GetValidator", ctx, valAddr).Return(validators[i], nil).Once()
			fn(delegation)
		}
	}).Once()

	// Also mock getIndividualDelegations call for this selector (since count > 1)
	sk.On("IterateDelegatorDelegations", ctx, multiDelegationSelector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for i, delegation := range delegations {
			valAddr, _ := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			sk.On("GetValidator", ctx, valAddr).Return(validators[i], nil).Once()
			fn(delegation)
		}
	}).Once()

	res, err := querier.SelectionsTo(ctx, &types.QuerySelectionsToRequest{ReporterAddress: reporterAddr.String()})
	fmt.Println(res)
	require.NoError(t, err)
	require.Equal(t, len(res.Selections), 11) // 10 single-delegation + 1 multi-delegation
	require.Equal(t, reporterAddr.String(), res.Reporter)
	fmt.Println(res)

	// Verify that each selection has the expected values
	// Convert selectorAddrs to strings for easier comparison
	expectedSelectors := make(map[string]bool)
	for _, addr := range selectorAddrs {
		expectedSelectors[addr.String()] = true
	}
	// Add the multi-delegation selector
	expectedSelectors[multiDelegationSelector.String()] = true

	var foundMultiDelegationSelector bool
	for _, selection := range res.Selections {
		require.True(t, expectedSelectors[selection.Selector], "Unexpected selector address: %s", selection.Selector)

		if selection.Selector == multiDelegationSelector.String() {
			// Verify multi-delegation selector
			foundMultiDelegationSelector = true
			require.Equal(t, uint64(3), selection.DelegationsCount)
			require.Equal(t, math.NewInt(1000*1e6), selection.DelegationsTotal) // 500M + 300M + 200M
			require.Len(t, selection.IndividualDelegations, 3)

			// Verify individual delegations have correct amounts
			expectedAmounts := []math.Int{
				math.NewInt(500 * 1e6),
				math.NewInt(300 * 1e6),
				math.NewInt(200 * 1e6),
			}
			expectedValidators := []string{
				sdk.ValAddress(val1Addr).String(),
				sdk.ValAddress(val2Addr).String(),
				sdk.ValAddress(val3Addr).String(),
			}

			// Check that all expected delegations are present
			actualAmounts := make(map[string]math.Int)
			for _, indivDel := range selection.IndividualDelegations {
				actualAmounts[indivDel.ValidatorAddress] = indivDel.Amount
			}

			for i, expectedVal := range expectedValidators {
				actualAmount, found := actualAmounts[expectedVal]
				require.True(t, found, "Expected validator %s not found in individual delegations", expectedVal)
				require.Equal(t, expectedAmounts[i], actualAmount, "Wrong amount for validator %s", expectedVal)
			}
		} else {
			// Verify single-delegation selectors
			require.Equal(t, uint64(1), selection.DelegationsCount)
			require.Equal(t, math.NewInt(1000*1e6), selection.DelegationsTotal)
			require.NotEmpty(t, selection.IndividualDelegations)
		}

		// Remove from expected to ensure no duplicates
		delete(expectedSelectors, selection.Selector)
	}

	// Ensure all selectors were found
	require.Empty(t, expectedSelectors, "Some selectors were not returned")
	require.True(t, foundMultiDelegationSelector, "Multi-delegation selector not found in results")
}
