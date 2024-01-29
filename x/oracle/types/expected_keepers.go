package types

import (
	context "context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type RegistryKeeper interface {
	// Methods imported from registry should be defined here
	Spec(ctx sdk.Context, queryType string) []byte
}
type StakingKeeper interface {
	// Methods imported from staking should be defined here
	GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error)
	Validator(ctx context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error)
	GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]stakingtypes.Delegation, error)
	GetLastTotalPower(ctx context.Context) (math.Int, error)
}

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

// DistributionKeeper defines the expected interface needed to retrieve community pool funds.
type DistrKeeper interface {
	AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error
}
