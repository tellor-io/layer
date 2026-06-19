package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func mockSelectorBondedStake(sk *mocks.StakingKeeper, ctx sdk.Context, selector sdk.AccAddress, valAddr sdk.ValAddress, tokens math.Int) {
	delegations := []stakingtypes.Delegation{
		{
			DelegatorAddress: selector.String(),
			ValidatorAddress: valAddr.String(),
			Shares:           tokens.ToLegacyDec(),
		},
	}
	sk.On("IterateDelegatorDelegations", ctx, selector, mock.AnythingOfType("func(types.Delegation) bool")).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(stakingtypes.Delegation) bool)
		for _, delegation := range delegations {
			if fn(delegation) {
				return
			}
		}
	})
}

func TestReporterPotentialStake(t *testing.T) {
	k, sk, _, _, _, ctx, _ := setupKeeper(t)
	ctx = ctx.WithBlockHeight(10).WithBlockTime(time.Now())

	valAddr := sdk.ValAddress(sample.AccAddressBytes())
	val := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
		Status:            stakingtypes.Bonded,
		Tokens:            math.NewInt(1000),
		DelegatorShares:   math.NewInt(1000).ToLegacyDec(),
		MinSelfDelegation: math.OneInt(),
	}
	sk.On("GetValidator", ctx, valAddr).Return(val, nil)

	reporterAddr := sample.AccAddressBytes()
	otherReporter := sample.AccAddressBytes()
	selfSelector := reporterAddr
	lockedSelector := sample.AccAddressBytes()
	leavingSelector := sample.AccAddressBytes()
	incomingSelector := sample.AccAddressBytes()

	// active self stake counts
	require.NoError(t, k.Selectors.Set(ctx, selfSelector.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))
	mockSelectorBondedStake(sk, ctx, selfSelector, valAddr, math.NewInt(20))

	// dispute-locked selectors count: their stake returns when the lock expires
	locked := types.NewSelection(reporterAddr.Bytes(), 1)
	locked.LockedUntilTime = ctx.BlockTime().Add(time.Hour)
	require.NoError(t, k.Selectors.Set(ctx, lockedSelector.Bytes(), locked))
	mockSelectorBondedStake(sk, ctx, lockedSelector, valAddr, math.NewInt(7))

	// selectors with a pending switch away are excluded
	require.NoError(t, k.Selectors.Set(ctx, leavingSelector.Bytes(), types.NewSelection(reporterAddr.Bytes(), 1)))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(reporterAddr.Bytes(), leavingSelector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  otherReporter.Bytes(),
		UnlockBlock: 100,
	}))

	// selectors with a pending switch into the reporter count
	require.NoError(t, k.Selectors.Set(ctx, incomingSelector.Bytes(), types.NewSelection(otherReporter.Bytes(), 1)))
	require.NoError(t, k.OutgoingPendingSwitches.Set(ctx, collections.Join(otherReporter.Bytes(), incomingSelector.Bytes()), types.PendingSwitchEntry{
		ToReporter:  reporterAddr.Bytes(),
		UnlockBlock: 100,
	}))
	require.NoError(t, k.IncomingPendingSwitchIdx.Set(ctx, collections.Join(reporterAddr.Bytes(), incomingSelector.Bytes()), otherReporter.Bytes()))
	mockSelectorBondedStake(sk, ctx, incomingSelector, valAddr, math.NewInt(5))

	total, err := k.ReporterPotentialStake(ctx, reporterAddr)
	require.NoError(t, err)
	// 20 (self) + 7 (locked) + 5 (pending incoming); the leaving selector's
	// stake is excluded
	require.Equal(t, math.NewInt(32), total)

	// state must not be mutated by the read
	sel, err := k.Selectors.Get(ctx, lockedSelector.Bytes())
	require.NoError(t, err)
	require.Equal(t, locked.LockedUntilTime.Unix(), sel.LockedUntilTime.Unix())
}

func TestReporterPotentialStakeNoSelectors(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	total, err := k.ReporterPotentialStake(ctx, sample.AccAddressBytes())
	require.NoError(t, err)
	require.True(t, total.IsZero())
}
