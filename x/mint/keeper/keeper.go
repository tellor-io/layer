package keeper

import (
	"context"
	"fmt"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

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

	Minter collections.Item[types.Minter]

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

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:           cdc,
		storeService:  storeService,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,

		Minter:    collections.NewItem(sb, collections.NewPrefix(0), "minter", codec.CollValue[types.Minter](cdc)),
		authority: authority,
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
	return k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// func (k Keeper) CalculateExtraRewards(ctx context.Context, current, previous time.Time) (sdk.Coin, error) {
//     if current.Before(previous) {
//         return sdk.Coin{}, fmt.Errorf("current time %v cannot be before previous time %v", current, previous)
//     }

//     // TODO:
// 	// params := k.GetParams(ctx)
// 	// dailyExtraRewardRate := params.ExtraRewardRate

//     dailyExtraRewards := types.DailyMintRate

//     timeElapsedMs := current.Sub(previous).Milliseconds()
//     rewardAmount := dailyExtraRewards * timeElapsedMs / 86400000

//     return sdk.NewCoin(layer.BondDenom, cosmosmath.NewInt(rewardAmount)), nil
// }

func (k Keeper) SendExtraRewards(ctx context.Context, coins sdk.Coins) error {
	if coins.Empty() {
		return nil
	}
	coinsAmt := coins.AmountOf(layer.BondDenom)
	// return nil if amt is zero to avoid constructing invalid transactions
	if coinsAmt.IsZero() {
		return nil
	}
	// get balance
	balance := k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ExtraRewardsPool), layer.BondDenom)
	if balance.IsZero() {
		return nil
	}

	// only send if we have enough balance so the minimum/rate is the the TBR mint rate
	if balance.Amount.LT(coinsAmt) {
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
	moduleAddress := authtypes.NewModuleAddressOrBech32Address(types.ExtraRewardsPool)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, threequarters.Add(quarter))))
	return k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}

// PreMintingSendExtraRewards sends extra rewards from the extra rewards pool before minting new TBR coins is initiated.
// Use same rate as TBR minting rate.
func (k Keeper) PreMintingSendExtraRewards(ctx context.Context) error {
	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()

	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}

	coin, err := minter.CalculateBlockProvision(currentTime, *minter.PreviousBlockTime)
	if err != nil {
		return err
	}
	amt := coin.Amount

	if amt.IsZero() {
		return nil
	}
	// get balance
	balance := k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ExtraRewardsPool), layer.BondDenom)
	if balance.IsZero() {
		return nil
	}

	// only send if we have enough balance so the minimum/rate is the the TBR mint rate
	if balance.Amount.LT(amt) {
		return nil
	}
	quarter := amt.QuoRaw(4)
	threequarters := amt.Sub(quarter)
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
	moduleAddress := authtypes.NewModuleAddressOrBech32Address(types.ExtraRewardsPool)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, threequarters.Add(quarter))))
	err = k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
	if err != nil {
		return err
	}
	minter.PreviousBlockTime = &currentTime
	return k.Minter.Set(ctx, minter)
}
