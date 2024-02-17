package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

func TestAllocateTokensToValidatorWithCommission(t *testing.T) {
	k, _, _, ctx := setupMsgServer(t)
	// create reporter with 50% commission
	reporterAcc := sdk.AccAddress([]byte("reporter"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAcc.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(ctx, reporterAcc, reporter)
	require.NoError(t, err)
	// allocate tokens
	tokens := sdk.DecCoins{
		{Denom: sdk.DefaultBondDenom, Amount: math.LegacyNewDec(10)},
	}
	require.NoError(t, k.AllocateTokensToReporter(ctx, reporterAcc, tokens))

	// check commission
	expected := sdk.DecCoins{
		{Denom: sdk.DefaultBondDenom, Amount: math.LegacyNewDec(5)},
	}

	valCommission, err := k.ReportersAccumulatedCommission.Get(ctx, sdk.ValAddress(reporterAcc))
	require.NoError(t, err)
	require.Equal(t, expected, valCommission.Commission)

	// check current rewards
	currentRewards, err := k.ReporterCurrentRewards.Get(ctx, sdk.ValAddress(reporterAcc))
	require.NoError(t, err)
	require.Equal(t, expected, currentRewards.Rewards)
}
