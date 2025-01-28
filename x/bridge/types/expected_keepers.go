package types

import (
	context "context"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	// Methods imported from staking should be defined here
	GetValidators(ctx context.Context, maxRetrieve uint32) ([]stakingtypes.Validator, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	TotalBondedTokens(ctx context.Context) (math.Int, error)
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amts sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amts sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type OracleKeeper interface {
	GetAggregateBefore(ctx context.Context, queryId []byte, timestampBefore time.Time) (aggregate *oracletypes.Aggregate, timestamp time.Time, err error)
	GetAggregateByTimestamp(ctx context.Context, queryId []byte, timestamp uint64) (oracletypes.Aggregate, error)
	GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetAggregatedReportsByHeight(ctx context.Context, height uint64) []oracletypes.Aggregate
	SetAggregate(ctx context.Context, report *oracletypes.Aggregate) error
}

type ReporterKeeper interface {
	TotalReporterPower(ctx context.Context) (math.Int, error)
}
