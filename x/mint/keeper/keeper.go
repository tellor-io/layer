package keeper

import (
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Keeper of the mint store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper types.BankKeeper
}

// NewKeeper creates a new mint Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
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
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetMinter returns the minter.
func (k Keeper) GetMinter(ctx sdk.Context) (minter types.Minter) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get(types.MinterKey())
	if b == nil {
		panic("stored minter should not have been nil")
	}

	k.cdc.MustUnmarshal(b, &minter)
	return minter
}

// SetMinter sets the minter.
func (k Keeper) SetMinter(ctx sdk.Context, minter types.Minter) {
	store := ctx.KVStore(k.storeKey)
	b := k.cdc.MustMarshal(&minter)
	store.Set(types.MinterKey(), b)
}

// MintCoins implements an alias call to the underlying bank keeper's
// MintCoins.
func (k Keeper) MintCoins(ctx sdk.Context, newCoins sdk.Coins) error {
	if newCoins.Empty() {
		return nil
	}
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

func (k Keeper) SendInflationaryRewards(ctx sdk.Context, coins sdk.Coins) error {
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
