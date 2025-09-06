package mint_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestBeginBlocker(t *testing.T) {
	require := require.New(t)

	k, ak, bk, ctx := keeper.MintKeeper(t)
	ctx = ctx.WithBlockTime(time.Now().UTC())
	extraRewardsAddr := authtypes.NewModuleAddress(types.ExtraRewardsPool)
	ak.On("GetModuleAddress", types.ExtraRewardsPool).Return(extraRewardsAddr)
	bk.On("GetBalance", mock.Anything, mock.Anything, mock.Anything).Return(sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())).Maybe()

	err := mint.BeginBlocker(ctx, k)
	require.Error(err)

	// set minter
	minter := types.DefaultMinter()
	currentTime := ctx.BlockTime()
	minter.PreviousBlockTime = &currentTime
	require.NoError(k.Minter.Set(ctx, minter))

	// future time
	ctx = ctx.WithBlockTime(time.Now().UTC().Add(5 * time.Second))

	timeElapsed := ctx.BlockTime().Sub(*minter.PreviousBlockTime).Milliseconds()
	mintAmount := types.DailyMintRate * timeElapsed / types.MillisecondsInDay

	bk.On("MintCoins", ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", math.NewInt(mintAmount)))).Return(nil)
	bk.On("InputOutputCoins", ctx, mock.Anything, mock.Anything).Return(nil)
	err = mint.BeginBlocker(ctx, k)
	require.Nil(err)

	// Initilize minter
	minter.Initialized = true
	require.NoError(k.Minter.Set(ctx, minter))
	err = mint.BeginBlocker(ctx, k)
	require.Nil(err)

	err = mint.BeginBlocker(ctx, k)
	require.Nil(err)

	minter, err = k.Minter.Get(ctx)
	require.Nil(err)
	require.Equal(minter.PreviousBlockTime.Unix(), ctx.BlockTime().Unix())
}

func TestMintBlockProvision(t *testing.T) {
	require := require.New(t)

	k, _, bk, ctx := keeper.MintKeeper(t)
	minter := types.DefaultMinter()
	minter.Initialized = true

	// prev block time is 0
	require.NoError(k.Minter.Set(ctx, minter))
	err := mint.MintBlockProvision(ctx, k, time.Unix(1000, 0), minter)
	require.Nil(err)

	// prev block time 5 sec ago
	time5SecAgo := time.Now().Add(-5 * time.Second)
	minter.PreviousBlockTime = &time5SecAgo
	require.NoError(k.Minter.Set(ctx, minter))
	expectedAmt := math.NewInt(types.DailyMintRate * 5 * 1000 / types.MillisecondsInDay)
	bk.On("MintCoins", ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("loya", expectedAmt))).Return(nil)
	bk.On("InputOutputCoins", ctx, mock.Anything, mock.Anything).Return(nil)

	err = mint.MintBlockProvision(ctx, k, time.Now(), minter)
	require.Nil(err)
}

func TestSetPreviousBlockTime(t *testing.T) {
	require := require.New(t)

	k, _, _, ctx := keeper.MintKeeper(t)
	minter := types.DefaultMinter()
	minter.Initialized = true
	require.NoError(k.Minter.Set(ctx, minter))

	time1 := time.Unix(1000, 0)
	time2 := time.Unix(2000, 0)
	ctx = ctx.WithBlockTime(time1)
	err := mint.SetPreviousBlockTime(ctx, k, time2)
	require.Nil(err)

	minter, err = k.Minter.Get(ctx)
	require.Nil(err)
	require.Equal(minter.PreviousBlockTime.Unix(), time2.Unix())
}

func TestBeginBlockerWithExtraRewards(t *testing.T) {
	require := require.New(t)

	k, ak, bk, ctx := keeper.MintKeeper(t)

	ctx = ctx.WithBlockTime(time.Now().UTC())
	// Setup extra rewards pool address
	extraRewardsAddr := authtypes.NewModuleAddress(types.ExtraRewardsPool)
	ak.On("GetModuleAddress", types.ExtraRewardsPool).Return(extraRewardsAddr)

	// BeginBlocker calls SendExtraRewards even when minter not initialized
	minter := types.DefaultMinter()
	currentTime := ctx.BlockTime()
	minter.PreviousBlockTime = &currentTime
	require.NoError(k.Minter.Set(ctx, minter))

	// Setup extra rewards params
	extraParams := types.ExtraRewardParams{
		DailyExtraRewards: types.DailyMintRate,
		PreviousBlockTime: nil, // First time
		BondDenom:         types.DefaultBondDenom,
	}
	require.NoError(k.ExtraRewardParams.Set(ctx, extraParams))

	// Mock balance check - pool has zero balance
	bk.On("GetBalance", ctx, extraRewardsAddr, types.DefaultBondDenom).Return(sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())).Once()

	input := banktypes.NewInput(extraRewardsAddr, sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, math.ZeroInt())))
	outputs := []banktypes.Output{}
	bk.On("InputOutputCoins", ctx, input, outputs).Return(nil).Once()

	err := mint.BeginBlocker(ctx, k)
	require.NoError(err)

	// Verify extra rewards params were updated
	updatedParams, err := k.ExtraRewardParams.Get(ctx)
	require.NoError(err)
	require.NotNil(updatedParams.PreviousBlockTime)

	// BeginBlocker with sufficient extra rewards balance
	// Advance time by 1 day
	newTime := currentTime.Add(24 * time.Hour)
	ctx = ctx.WithBlockTime(newTime)

	// Set previous time for extra rewards
	extraParams.PreviousBlockTime = &currentTime
	require.NoError(k.ExtraRewardParams.Set(ctx, extraParams))

	// Calculate expected extra reward amount (1 day worth)
	expectedExtraReward := math.NewInt(types.DailyMintRate)
	poolBalance := sdk.NewCoin(types.DefaultBondDenom, expectedExtraReward.MulRaw(2))

	bk.On("GetBalance", ctx, extraRewardsAddr, types.DefaultBondDenom).Return(poolBalance).Once()
	banktypesInput := banktypes.NewInput(extraRewardsAddr, sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedExtraReward)))
	banktypesOutputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedExtraReward.QuoRaw(4).MulRaw(3))),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedExtraReward.QuoRaw(4))),
		},
	}
	bk.On("InputOutputCoins", ctx, banktypesInput, banktypesOutputs).Return(nil).Once()

	// Initialize minter for regular minting
	minter.Initialized = true
	minter.PreviousBlockTime = &currentTime
	require.NoError(k.Minter.Set(ctx, minter))

	bk.On("MintCoins", ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, math.NewInt(types.DailyMintRate)))).Return(nil).Once()
	input = banktypes.NewInput(authtypes.NewModuleAddress(types.ModuleName), sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, math.NewInt(types.DailyMintRate))))
	outputs = []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, math.NewInt(types.DailyMintRate).QuoRaw(4).MulRaw(3))),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, math.NewInt(types.DailyMintRate).QuoRaw(4))),
		},
	}
	bk.On("InputOutputCoins", ctx, input, outputs).Return(nil).Once()

	err = mint.BeginBlocker(ctx, k)
	require.NoError(err)

	// Verify both minter and extra rewards times were updated
	updatedMinter, err := k.Minter.Get(ctx)
	require.NoError(err)
	require.Equal(newTime, *updatedMinter.PreviousBlockTime)

	updatedExtraParams, err := k.ExtraRewardParams.Get(ctx)
	require.NoError(err)
	require.Equal(newTime, *updatedExtraParams.PreviousBlockTime)
}

func TestExtraRewardsIndependence(t *testing.T) {
	require := require.New(t)

	k, ak, bk, ctx := keeper.MintKeeper(t)
	ctx = ctx.WithBlockTime(time.Now().UTC())

	require.NoError(k.Minter.Set(ctx, types.DefaultMinter())) // Minter not initialized

	currentTime := ctx.BlockTime()

	previousTime := currentTime.Add(-12 * time.Hour) // 12 hours ago
	extraParams := types.ExtraRewardParams{
		DailyExtraRewards: types.DailyMintRate,
		PreviousBlockTime: &previousTime,
		BondDenom:         types.DefaultBondDenom,
	}
	require.NoError(k.ExtraRewardParams.Set(ctx, extraParams))

	// Calculate expected reward for 12 hours
	expectedReward := math.NewInt(types.DailyMintRate / 2) // Half day worth
	poolBalance := sdk.NewCoin(types.DefaultBondDenom, expectedReward)

	// Setup extra rewards pool address
	extraRewardsAddr := authtypes.NewModuleAddress(types.ExtraRewardsPool)
	ak.On("GetModuleAddress", types.ExtraRewardsPool).Return(extraRewardsAddr)
	bk.On("GetBalance", ctx, extraRewardsAddr, types.DefaultBondDenom).Return(poolBalance).Once()
	input := banktypes.NewInput(extraRewardsAddr, sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedReward)))
	outputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedReward.QuoRaw(4).MulRaw(3))),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(types.DefaultBondDenom, expectedReward.QuoRaw(4))),
		},
	}
	bk.On("InputOutputCoins", ctx, input, outputs).Return(nil).Once()

	// No regular minting should occur since minter not initialized
	err := mint.BeginBlocker(ctx, k)
	require.NoError(err)

	// Verify extra rewards time was updated but minter wasn't
	updatedExtraParams, err := k.ExtraRewardParams.Get(ctx)
	require.NoError(err)
	require.Equal(currentTime, *updatedExtraParams.PreviousBlockTime)

	// Minter should still be uninitialized
	updatedMinter, err := k.Minter.Get(ctx)
	require.NoError(err)
	require.False(updatedMinter.Initialized)
}

func BenchmarkMintBeginBlocker(b *testing.B) {
	k, _, _, ctx := keeper.MintKeeper(b)

	minter := types.DefaultMinter()
	minter.Initialized = true
	require.NoError(b, k.Minter.Set(ctx, minter))

	ctx = ctx.WithBlockTime(time.Unix(1000, 0))

	b.ResetTimer() // Start timing from here

	for i := 0; i < b.N; i++ {
		err := mint.BeginBlocker(ctx, k)
		require.NoError(b, err)
	}
}
