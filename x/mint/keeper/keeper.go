package keeper

import (
	"context"
	"fmt"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Keeper of the mint store
type Keeper struct {
	cdc           codec.BinaryCodec
	storeService  storetypes.KVStoreService
	Schema        collections.Schema
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper

	Minter            collections.Item[types.Minter]
	ExtraRewardParams collections.Item[types.ExtraRewardParams]

	authority string
}

// NewKeeper creates a new mint Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}
	// Ensure the mint module account has been set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the mint module account has not been set")
	}
	// Ensure the mintToOracle account has been set
	if addr := accountKeeper.GetModuleAddress(types.TimeBasedRewards); addr == nil {
		panic("the mintToOracle account has not been set")
	}
	// Ensure the extra rewards pool account has been set
	if addr := accountKeeper.GetModuleAddress(types.ExtraRewardsPool); addr == nil {
		panic("the extra rewards pool account has not been set")
	}
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:           cdc,
		storeService:  storeService,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,

		Minter:            collections.NewItem(sb, collections.NewPrefix(0), "minter", codec.CollValue[types.Minter](cdc)),
		ExtraRewardParams: collections.NewItem(sb, collections.NewPrefix(1), "extra_reward_params", codec.CollValue[types.ExtraRewardParams](cdc)),
		authority:         authority,
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}

// MintCoins implements an alias call to the underlying bank keeper's
// MintCoins.
func (k Keeper) MintCoins(ctx context.Context, newCoins sdk.Coins) error {
	if newCoins.Empty() {
		return nil
	}
	k.Logger(ctx).Info("minting tbr", "coins", newCoins)
	// emit event with amount and destination
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"mint_coins",
			sdk.NewAttribute("amount", newCoins.String()),
			sdk.NewAttribute("destination", types.ModuleName),
		),
	})
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

func (k Keeper) SendInflationaryRewards(ctx context.Context, coins sdk.Coins) error {
	if coins.Empty() {
		return nil
	}
	// declare coins.AmountOf(layer.BondDenom) to optimize gas
	coinsAmt := coins.AmountOf(layer.BondDenom)
	// return nil if amt is zero to avoid constructing invalid transactions
	if coinsAmt.IsZero() {
		return nil
	}
	quarter := coinsAmt.QuoRaw(4)
	threequarters := coinsAmt.Sub(quarter)
	outputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(layer.BondDenom, threequarters)),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(layer.BondDenom, quarter)),
		},
	}
	moduleAddress := authtypes.NewModuleAddressOrBech32Address(types.ModuleName)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, threequarters.Add(quarter))))
	err := k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalRewardCoins := sdk.NewCoins(sdk.NewCoin(layer.BondDenom, coinsAmt))
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"inflationary_rewards_distributed",
			sdk.NewAttribute("total_amount", totalRewardCoins.String()),
		),
	})
	return nil
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetExtraRewardRateParams(ctx context.Context) types.ExtraRewardParams {
	params, err := k.ExtraRewardParams.Get(ctx)
	if err != nil {
		return types.ExtraRewardParams{BondDenom: types.DefaultBondDenom}
	}
	return params
}

// SendExtraRewards sends extra rewards from the extra rewards pool before minting new TBR coins is initiated.
// Use same rate as TBR minting rate.
func (k Keeper) SendExtraRewards(ctx context.Context) error {
	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()

	rewardParams := k.GetExtraRewardRateParams(ctx)
	previousBlockTime := rewardParams.PreviousBlockTime
	// set a new previous block time regardless of whether we can pay out rewards or not
	rewardParams.PreviousBlockTime = &currentTime
	if err := k.ExtraRewardParams.Set(ctx, rewardParams); err != nil {
		return err
	}

	if previousBlockTime == nil {
		return nil
	}
	// get balance
	moduleAddr := k.accountKeeper.GetModuleAddress(types.ExtraRewardsPool)
	balance := k.bankKeeper.GetBalance(ctx, moduleAddr, rewardParams.BondDenom)
	if balance.IsZero() {
		return nil
	}

	dailyExtraRewardRate := rewardParams.DailyExtraRewards
	if dailyExtraRewardRate == 0 {
		dailyExtraRewardRate = types.DailyMintRate
	}

	timeElapsed := currentTime.Sub(*previousBlockTime).Milliseconds()
	if timeElapsed < 0 {
		k.Logger(ctx).Error("extra rewards time elapsed is negative", "time_elapsed", timeElapsed)
		return nil
	}
	rewardAmount := dailyExtraRewardRate * timeElapsed / types.MillisecondsInDay
	rewardAmountInt := math.NewInt(rewardAmount)
	// only send if we have enough balance so the minimum/rate is the the TBR mint rate
	if rewardAmountInt.IsZero() || balance.Amount.LT(rewardAmountInt) {
		return nil
	}

	quarter := rewardAmountInt.QuoRaw(4)
	threequarters := rewardAmountInt.Sub(quarter)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	totalRewardCoins := sdk.NewCoins(sdk.NewCoin(rewardParams.BondDenom, rewardAmountInt))

	outputs := []banktypes.Output{
		{
			Address: authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(rewardParams.BondDenom, threequarters)),
		},
		{
			Address: authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(rewardParams.BondDenom, quarter)),
		},
	}
	moduleAddress := authtypes.NewModuleAddressOrBech32Address(types.ExtraRewardsPool)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(rewardParams.BondDenom, threequarters.Add(quarter))))
	err := k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("minting extra rewards", "coins", totalRewardCoins)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"extra_rewards_distributed",
			sdk.NewAttribute("total_amount", totalRewardCoins.String()),
		),
	})
	return nil
}
