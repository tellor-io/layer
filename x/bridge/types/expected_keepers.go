package types

import (
	context "context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type StakingKeeper interface {
	// Methods imported from staking should be defined here
	GetValidators(ctx context.Context, maxRetrieve uint32) ([]stakingtypes.Validator, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
}

type SlashingKeeper interface {
	// Methods imported from slashing should be defined here
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amts sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amts sdk.Coins) error
	// Methods imported from bank should be defined here
}

type OracleKeeper interface {
	GetCurrentAggregateReport(ctx context.Context, queryId []byte) (aggregate *oracletypes.Aggregate, timestamp time.Time)
	GetAggregateBefore(ctx context.Context, queryId []byte, timestampBefore time.Time) (aggregate *oracletypes.Aggregate, timestamp time.Time, err error)
	GetAggregateByTimestamp(ctx sdk.Context, queryId []byte, timestamp time.Time) (aggregate *oracletypes.Aggregate, err error)
	GetTimestampBefore(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetTimestampAfter(ctx context.Context, queryId []byte, timestamp time.Time) (time.Time, error)
	GetAggregatedReportsByHeight(ctx context.Context, height int64) []oracletypes.Aggregate
	SetAggregate(ctx context.Context, report *oracletypes.Aggregate) error
}
