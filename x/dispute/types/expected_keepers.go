package types

import (
	context "context"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type SlashingKeeper interface {
	JailUntil(ctx context.Context, consAddr sdk.ConsAddress, jailTime time.Time) error
	GetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress) (slashingtypes.ValidatorSigningInfo, error)
	SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error
}

type StakingKeeper interface {
	ConsensusAddressCodec() address.Codec
	// Methods imported from staking should be defined here
	AddValidatorTokensAndShares(ctx context.Context, validator stakingtypes.Validator,
		tokensToAdd math.Int,
	) (valOut stakingtypes.Validator, addedShares math.LegacyDec, err error)
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool,
	) (newShares math.LegacyDec, err error)
	DeleteValidatorByPowerIndex(ctx context.Context, validator stakingtypes.Validator) error
	GetLastTotalPower(ctx context.Context) (math.Int, error)
	GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, err error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, err error)
	IterateDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress, cb func(delegation stakingtypes.Delegation) (stop bool)) error
	Jail(ctx context.Context, consAddr sdk.ConsAddress) error
	RemoveDelegation(ctx context.Context, delegation stakingtypes.Delegation) error
	RemoveValidatorTokens(ctx context.Context, validator stakingtypes.Validator, tokensToRemove math.Int) (stakingtypes.Validator, error)
	RemoveValidatorTokensAndShares(ctx context.Context, validator stakingtypes.Validator, sharesToRemove math.LegacyDec) (valOut stakingtypes.Validator, removedTokens math.Int, err error)
	SetDelegation(ctx context.Context, delegation stakingtypes.Delegation) error
	SetValidator(ctx context.Context, validator stakingtypes.Validator) error
	SetValidatorByPowerIndex(ctx context.Context, validator stakingtypes.Validator) error
	TokensFromConsensusPower(ctx context.Context, power int64) math.Int
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetSupply(ctx context.Context, denom string) sdk.Coin
	HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	InputOutputCoins(ctx context.Context, inputs banktypes.Input, outputs []banktypes.Output) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

type OracleKeeper interface {
	GetTotalTips(ctx sdk.Context) sdk.Coin
	GetUserTips(ctx sdk.Context, tipper sdk.AccAddress) oracletypes.UserTipTotal
}
