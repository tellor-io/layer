package types

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type SlashingKeeper interface {
	JailUntil(ctx sdk.Context, consAddr sdk.ConsAddress, jailTime time.Time)
	GetValidatorSigningInfo(ctx sdk.Context, address sdk.ConsAddress) (info slashingtypes.ValidatorSigningInfo, found bool)
	SetValidatorSigningInfo(ctx sdk.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo)
}

type StakingKeeper interface {
	// Methods imported from staking should be defined here
	AddValidatorTokensAndShares(ctx sdk.Context, validator stakingtypes.Validator,
		tokensToAdd math.Int,
	) (valOut stakingtypes.Validator, addedShares sdk.Dec)
	Delegate(ctx sdk.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool,
	) (newShares sdk.Dec, err error)
	DeleteValidatorByPowerIndex(ctx sdk.Context, validator stakingtypes.Validator)
	GetLastTotalPower(ctx sdk.Context) math.Int
	GetDelegation(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, found bool)
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	GetValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, found bool)
	IterateDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress, cb func(delegation stakingtypes.Delegation) (stop bool))
	Jail(ctx sdk.Context, consAddr sdk.ConsAddress)
	RemoveDelegation(ctx sdk.Context, delegation stakingtypes.Delegation) error
	RemoveValidatorTokens(ctx sdk.Context, validator stakingtypes.Validator, tokensToRemove math.Int) stakingtypes.Validator
	RemoveValidatorTokensAndShares(ctx sdk.Context, validator stakingtypes.Validator, sharesToRemove sdk.Dec) (valOut stakingtypes.Validator, removedTokens math.Int)
	SetDelegation(ctx sdk.Context, delegation stakingtypes.Delegation)
	SetValidator(ctx sdk.Context, validator stakingtypes.Validator)
	SetValidatorByPowerIndex(ctx sdk.Context, validator stakingtypes.Validator)
	TokensFromConsensusPower(ctx sdk.Context, power int64) math.Int
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetSupply(ctx sdk.Context, denom string) sdk.Coin
	HasBalance(ctx sdk.Context, addr sdk.AccAddress, amt sdk.Coin) bool
	InputOutputCoins(ctx sdk.Context, inputs []banktypes.Input, outputs []banktypes.Output) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

type OracleKeeper interface {
	GetTotalTips(ctx sdk.Context) sdk.Coin
	GetUserTips(ctx sdk.Context, tipper sdk.AccAddress) oracletypes.UserTipTotal
}
