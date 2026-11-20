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
	IsAggregateReport(ctx context.Context, report oracletypes.MicroReport) (bool, error)
	ValidateMicroReportExists(ctx context.Context, reporter sdk.AccAddress, meta_id uint64, query_id []byte) (*oracletypes.MicroReport, bool, error)
}

type ReporterKeeper interface {
	EscrowReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, power, height uint64, amt math.Int, queryId, hashId []byte) error
	JailReporter(ctx context.Context, reporterAddr sdk.AccAddress, jailDuration, reportBlockNumber uint64, disputeHashID []byte) error
	TotalReporterPower(ctx context.Context) (math.Int, error)
	FeefromReporterStake(ctx context.Context, reporterAddr sdk.AccAddress, amt math.Int, hashId []byte, isFirstRound bool) error
	DisputedDelegationTotal(ctx context.Context, hashId []byte) (math.Int, error)
	ReturnSlashedTokens(ctx context.Context, hashId []byte, extraReturn math.Int) (math.Int, math.Int, error)
	AddAmountToStake(ctx context.Context, acc sdk.AccAddress, amt math.Int) (math.Int, math.Int, error)
	Delegation(ctx context.Context, delegator sdk.AccAddress) (reportertypes.Selection, error)
	GetReporterTokensAtBlock(ctx context.Context, reporter []byte, blockNumber uint64) (math.Int, error)
	GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber uint64) (math.Int, error)
	GetDelegatorTokensFromReporterAtBlock(ctx context.Context, delegator, reporter []byte, blockNumber uint64) (math.Int, error)
	FeePaidFromStakeTotalByPayer(ctx context.Context, hashId []byte, payer sdk.AccAddress) (math.Int, error)
	FeeRefund(ctx context.Context, hashId []byte, payer sdk.AccAddress, amt math.Int) (math.Int, math.Int, error)
	UpdateJailedUntilOnFailedDispute(ctx context.Context, reporterAddr sdk.AccAddress, reportBlockNumber uint64, disputeHashID []byte) error
	GetSelectorForStake(ctx context.Context, selectorAddr sdk.AccAddress) (reportertypes.Selection, error)
	// ReporterAtHeight returns the reporter this selector was on at blockHeight.
	// liveAccurate is true when that reporter is already the live selection, which
	// is the case when the newest switch completed at or before blockHeight.
	// An open switch still resolves to the original reporter. It does not by itself
	// mean a switch was unfinished; use SelectorSwitchUnfinishedAt for that.
	// A missing selection row still resolves from history when blockHeight is before
	// the newest completion. Otherwise it returns collections.ErrNotFound: the answer
	// would be the deleted live reporter.
	ReporterAtHeight(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (sdk.AccAddress, bool, error)
	// SelectorSwitchUnfinishedAt reports whether a switch was in progress at blockHeight:
	// it began at or before blockHeight and had not completed by then.
	SelectorSwitchUnfinishedAt(ctx context.Context, selector sdk.AccAddress, blockHeight uint64) (bool, error)
	// PruneSelectorSwitches deletes completed switch rows no open dispute can query.
	// pruneHeight is the oldest open dispute block, or the current block when none are open.
	PruneSelectorSwitches(ctx context.Context, pruneHeight uint64, limit int) error
}
