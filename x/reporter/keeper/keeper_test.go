package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, sdk.Context, store.KVStoreService) {
	tb.Helper()
	return keepertest.ReporterKeeper(tb)
}

func TestKeeper(t *testing.T) {
	k, sk, bk, _, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
}

func TestGetAuthority(t *testing.T) {
	k, _, _, _, _, _ := setupKeeper(t)
	authority := k.GetAuthority()
	require.NotEmpty(t, authority)

	// convert to address and check if it's valid
	_, err := sdk.AccAddressFromBech32(authority)
	require.NoError(t, err)
}

func TestLogger(t *testing.T) {
	k, _, _, _, _, _ := setupKeeper(t)
	require.NotNil(t, k.Logger())
}

func TestGetDelegatorTokensAtBlock(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	delAddr, val1Address, val2Address := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(delAddr, 2)))

	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val1Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val2Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
	require.NoError(t, k.Report.Set(ctx, collections.Join(delAddr.Bytes(), ctx.BlockHeight()), types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: tokenOrigin1.Amount.Add(tokenOrigin2.Amount)}))

	tokens, err := k.GetDelegatorTokensAtBlock(ctx, delAddr, ctx.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, math.NewIntWithDecimal(2000, 6), tokens)
}

func TestGetReporterTokensAtBlock(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	tokens, err := k.GetReporterTokensAtBlock(ctx, reporter, ctx.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), tokens)

	require.NoError(t, k.Report.Set(ctx, collections.Join(reporter.Bytes(), ctx.BlockHeight()), types.DelegationsAmounts{Total: math.OneInt()}))

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, ctx.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, ctx.BlockHeight()+10)
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)
}

func TestTrackStakeChange(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	expiration := ctx.BlockTime().Add(1)
	err := k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	require.NoError(t, k.TrackStakeChange(ctx))

	expiration = ctx.BlockTime()
	err = k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	sk.On("TotalBondedTokens", ctx).Return(math.OneInt(), nil)
	require.NoError(t, k.TrackStakeChange(ctx))

	change, err := k.Tracker.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), change.Amount)
	require.Equal(t, expiration.Add(12*time.Hour), *change.Expiration)
}
