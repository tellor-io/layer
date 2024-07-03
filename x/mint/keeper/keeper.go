package keeper

import (
	"context"

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
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	Schema       collections.Schema
	bankKeeper   types.BankKeeper

	Minter collections.Item[types.Minter]
}

// NewKeeper creates a new mint Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
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
		cdc:          cdc,
		storeService: storeService,
		bankKeeper:   bankKeeper,

		Minter: collections.NewItem(sb, types.MinterKey, "minter", codec.CollValue[types.Minter](cdc)),
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
	quarter := coins.AmountOf(layer.BondDenom).QuoRaw(4)
	threequarters := coins.AmountOf(layer.BondDenom).Sub(quarter)
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
