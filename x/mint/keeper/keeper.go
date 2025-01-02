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

	// send 3/4 from mint to time based rewards
	mintModuleAddress := authtypes.NewModuleAddressOrBech32Address(types.ModuleName).String()
	timeBasedRewardsModuleAddress := authtypes.NewModuleAddressOrBech32Address(types.TimeBasedRewards).String()
	k.bankKeeper.SendCoinsFromModuleToModule(ctx, mintModuleAddress, timeBasedRewardsModuleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, threequarters)))

	// send 1/4 from mint to fee collector
	feeCollectorModuleAddress := authtypes.NewModuleAddressOrBech32Address(authtypes.FeeCollectorName).String()
	k.bankKeeper.SendCoinsFromModuleToModule(ctx, mintModuleAddress, feeCollectorModuleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, quarter)))

	return nil
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
