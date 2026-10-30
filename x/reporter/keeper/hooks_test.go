package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestBeforeDelegationCreated(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	delAddr, valAddr := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr))

	_, err := k.Selectors.Get(ctx, delAddr.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(delAddr, 1)))
	require.NoError(t, k.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr))

	selector, err := k.Selectors.Get(ctx, delAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(2), selector.DelegationsCount)
}

func TestBeforeDelegationRemoved(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	delAddr, valAddr := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes())
	reporterAddr := sample.AccAddressBytes()
	require.NoError(t, k.Hooks().BeforeDelegationRemoved(ctx, delAddr, valAddr))

	_, err := k.Selectors.Get(ctx, delAddr.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)
	// Non-selector removal must not flag anyone
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, has)

	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(reporterAddr, 1)))
	require.NoError(t, k.Hooks().BeforeDelegationRemoved(ctx, delAddr, valAddr))

	selector, err := k.Selectors.Get(ctx, delAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(0), selector.DelegationsCount)

	// A full undelegation only fires this hook, so it must invalidate the
	// reporter's cached stake the same way AfterDelegationModified does.
	has, err = k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.True(t, has)
	has, err = k.StakeRecalcFlag.Has(ctx, delAddr.Bytes())
	require.NoError(t, err)
	require.False(t, has, "flag belongs to the reporter, not the selector")
}

func TestAfterValidatorBonded_SetsLastValSetUpdateHeight(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	consAddr := sdk.ConsAddress(sample.AccAddressBytes())
	valAddr := sdk.ValAddress(sample.AccAddressBytes())

	ctx = ctx.WithBlockHeight(42)

	// Should set LastValSetUpdateHeight
	require.NoError(t, k.Hooks().AfterValidatorBonded(ctx, consAddr, valAddr))

	height, err := k.LastValSetUpdateHeight.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(42), height)
}

func TestAfterValidatorBeginUnbonding_SetsLastValSetUpdateHeight(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	consAddr := sdk.ConsAddress(sample.AccAddressBytes())
	valAddr := sdk.ValAddress(sample.AccAddressBytes())

	ctx = ctx.WithBlockHeight(99)

	// Should set LastValSetUpdateHeight
	require.NoError(t, k.Hooks().AfterValidatorBeginUnbonding(ctx, consAddr, valAddr))

	height, err := k.LastValSetUpdateHeight.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(99), height)
}

func TestAfterDelegationModified_SetsRecalcFlag(t *testing.T) {
	k, _, _, _, _, ctx, _ := setupKeeper(t)
	selectorAddr := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	valAddr := sdk.ValAddress(sample.AccAddressBytes())

	// Non-selector delegation should not set any flag
	require.NoError(t, k.Hooks().AfterDelegationModified(ctx, selectorAddr, valAddr))
	has, err := k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.False(t, has)

	// Register selector with reporter
	require.NoError(t, k.Selectors.Set(ctx, selectorAddr, types.NewSelection(reporterAddr, 1)))

	// Now delegation modification should set the recalc flag for reporter
	require.NoError(t, k.Hooks().AfterDelegationModified(ctx, selectorAddr, valAddr))

	has, err = k.StakeRecalcFlag.Has(ctx, reporterAddr.Bytes())
	require.NoError(t, err)
	require.True(t, has)
}
