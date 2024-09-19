package types

import (
	context "context"

	rktypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAccount(ctx context.Context, name string) sdk.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type RegistryKeeper interface {
	// Methods imported from registry should be defined here
	GetSpec(ctx context.Context, queryType string) (rktypes.DataSpec, error)
}

type ReporterKeeper interface {
	// Methods imported from reporter should be defined here
	ReporterStake(ctx context.Context, repAddress sdk.AccAddress) (math.Int, error)
	DivvyingTips(ctx context.Context, reporterAddr sdk.AccAddress, reward math.LegacyDec, height uint64) error
}

type RegistryHooks interface {
	AfterDataSpecUpdated(ctx context.Context, querytype string, dataspec rktypes.DataSpec) error
}
