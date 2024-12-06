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
	require.NoError(t, k.Hooks().BeforeDelegationRemoved(ctx, delAddr, valAddr))

	_, err := k.Selectors.Get(ctx, delAddr.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(delAddr, 1)))
	require.NoError(t, k.Hooks().BeforeDelegationRemoved(ctx, delAddr, valAddr))

	selector, err := k.Selectors.Get(ctx, delAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(0), selector.DelegationsCount)
}
