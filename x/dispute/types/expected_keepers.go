package types

import (
	context "context"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

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
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type OracleKeeper interface {
	GetTotalTips(ctx context.Context) (math.Int, error)
	GetUserTips(ctx context.Context, tipper sdk.AccAddress) (math.Int, error)
	GetTotalTipsAtBlock(ctx context.Context, blockNumber uint64) (math.Int, error)
	GetTipsAtBlockForTipper(ctx context.Context, blockNumber uint64, tipper sdk.AccAddress) (math.Int, error)
	FlagAggregateReport(ctx context.Context, report oracletypes.MicroReport) error
}

type ReporterKeeper interface {
	EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height uint64, amt math.Int, queryId, hashId []byte) error
	JailReporter(ctx context.Context, reporterAddr sdk.AccAddress, jailDuration uint64) error
	TotalReporterPower(ctx context.Context) (math.Int, error)
	FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte, isFirstRound bool) error
	ReturnSlashedTokens(ctx context.Context, amt math.Int, hashId []byte) (string, error)
	AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) error
	Delegation(ctx context.Context, delegator sdk.AccAddress) (reportertypes.Selection, error)
	GetReporterTokensAtBlock(ctx context.Context, reporter []byte, blockNumber uint64) (math.Int, error)
	GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber uint64) (math.Int, error)
	FeeRefund(ctx context.Context, hashId []byte, amt math.Int) error
	UpdateJailedUntilOnFailedDispute(ctx context.Context, reporterAddr sdk.AccAddress) error
	GetSelector(ctx context.Context, selectorAddr sdk.AccAddress) (reportertypes.Selection, error)
}
