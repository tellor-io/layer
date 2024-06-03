package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestAllocateTokensToReporterWithCommission(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	// create reporter with 50% commission
	reporterAcc := sdk.AccAddress([]byte("reporter"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAcc.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(ctx, reporterAcc, reporter)
	require.NoError(t, err)
	// allocate tokens
	tokens := sdk.DecCoins{
		{Denom: types.Denom, Amount: math.LegacyNewDec(10)},
	}
	require.NoError(t, k.AllocateTokensToReporter(ctx, reporterAcc.Bytes(), tokens))

	// check commission
	expected := sdk.DecCoins{
		{Denom: types.Denom, Amount: math.LegacyNewDec(5)},
	}

	repCommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterAcc.Bytes())
	require.NoError(t, err)
	require.Equal(t, expected, repCommission.Commission)

	// check current rewards
	currentRewards, err := k.ReporterCurrentRewards.Get(ctx, reporterAcc.Bytes())
	require.NoError(t, err)
	require.Equal(t, expected, currentRewards.Rewards)
}

func TestAllocateTokensToManyReporters(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(ctx, reporterAccI, reporter)
	require.NoError(t, err)
	// create second reporter with 0% commission
	reporterAccII := sdk.AccAddress([]byte("reporter2"))
	commission = types.NewCommissionWithTime(math.LegacyNewDec(0), math.LegacyNewDec(0),
		math.LegacyNewDec(0), time.Time{})
	reporter = types.NewOracleReporter(reporterAccII.String(), math.NewInt(100), &commission)
	err = k.Reporters.Set(ctx, reporterAccII, reporter)
	require.NoError(t, err)

	// assert initial state: zero outstanding rewards, zero commission, zero current rewards
	_, err = k.ReporterOutstandingRewards.Get(ctx, reporterAccI.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = k.ReporterOutstandingRewards.Get(ctx, reporterAccII.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = k.ReportersAccumulatedCommission.Get(ctx, reporterAccI.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = k.ReportersAccumulatedCommission.Get(ctx, reporterAccII.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = k.ReporterCurrentRewards.Get(ctx, reporterAccI.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound) // require no rewards

	_, err = k.ReporterCurrentRewards.Get(ctx, reporterAccII.Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound) // require no rewards

	require.NoError(t, k.AllocateTokensToReporter(ctx, reporterAccI.Bytes(), sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(50)}}))
	require.NoError(t, k.AllocateTokensToReporter(ctx, reporterAccII.Bytes(), sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(50)}}))

	// 100 outstanding rewards
	repIOutstandingRewards, err := k.ReporterOutstandingRewards.Get(ctx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(50)}}, repIOutstandingRewards.Rewards)

	repIIOutstandingRewards, err := k.ReporterOutstandingRewards.Get(ctx, reporterAccII.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(50)}}, repIIOutstandingRewards.Rewards)

	// 50% commission for first reporter
	repICommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(25)}}, repICommission.Commission)

	// zero commission for second reporter
	repIICommission, err := k.ReportersAccumulatedCommission.Get(ctx, reporterAccII.Bytes())
	require.NoError(t, err)
	require.True(t, repIICommission.Commission.IsZero())

	// just staking.proportional for first reporter less commission = (0.5 * 100%) * 100 / 2 = 25.00
	repICurrentRewards, err := k.ReporterCurrentRewards.Get(ctx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(25)}}, repICurrentRewards.Rewards)

	// reporter reward + staking.proportional for second reporter = (0.5 * (100%)) * 100 = 50
	repIICurrentRewards, err := k.ReporterCurrentRewards.Get(ctx, reporterAccII.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(50)}}, repIICurrentRewards.Rewards)
}

func TestCalculateRewardsBasic(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// delegation mock
	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// hooks
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// historical count should be 2 (once for reporter init, once for delegation init)
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)
	require.Equal(t, 2, getRepHistoricalReferenceCount(k, sdkCtx))

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// historical count should be 2 still
	require.Equal(t, 2, getRepHistoricalReferenceCount(k, sdkCtx))

	// calculate delegation rewards
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// // allocate some rewards
	initial := int64(10)
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial)}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// // end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)
	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 2)}}, rewards)

	// commission should be the other half
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 2)}}, repCommission.Commission)
}

func TestCalculateRewardsAfterSlash(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// slash the reporter by 50% (simulated with manual calls; we assume the reporter is bonded)
	reporter, burned := SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	// increase block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// allocate some rewards
	initial := math.NewInt(10)
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial)}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial.QuoRaw(2))}}, rewards)

	// commission should be the other half
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial.QuoRaw(2))}},
		repCommission.Commission)
}

func TestCalculateRewardsAfterManySlashes(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// slash the reporter by 50% (simulated with manual calls; we assume the reporter is bonded)
	reporter, burned := SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	// increase block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// // allocate some rewards
	initial := sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial)}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// slash the reporter by 50% (simulated with manual calls; we assume the reporter is bonded)
	reporter, burned = SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(2, 1), reporter, k)

	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	// increase block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial)}}, rewards)

	// commission should be the other half
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDecFromInt(initial)}}, repCommission.Commission)
}

func TestCalculateRewardsMultiDelegator(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)

	// create reporter with 50% commission
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	initial := int64(20)
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial)}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// second delegation
	delegatorII := sdk.AccAddress([]byte("delegator2"))
	delegationII := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.BeforeDelegationCreated(sdkCtx, reporter)
	require.NoError(t, err)
	err = k.Delegators.Set(sdkCtx, delegatorII, delegationII)
	require.NoError(t, err)
	// call necessary hooks to update a delegation
	// end period
	err = k.AfterDelegationModified(sdkCtx, delegatorII, reporterAccI.Bytes(), delegationII.Amount)
	require.NoError(t, err)

	// update reporter with new total tokens from the second delegation
	reporter.TotalTokens = reporter.TotalTokens.Add(delegationII.Amount)
	err = k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards for del0 should be 3/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial * 3 / 4)}}, rewards)

	// calculate delegation rewards for del2
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII, delegationII, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial * 1 / 4)}}, rewards)

	// commission should be equal to initial (50% twice)
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial)}}, repCommission.Commission)
}

func TestWithdrawDelegationRewardsBasic(t *testing.T) {
	k, _, bk, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	initial := sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)
	tokens := sdk.DecCoins{sdk.NewDecCoin(types.Denom, initial)}

	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// historical count should be 2 (initial + latest for delegation)
	require.Equal(t, 2, getRepHistoricalReferenceCount(k, sdkCtx))

	// withdraw rewards (the bank keeper should be called with the right amount of tokens to transfer)
	expRewards := sdk.Coins{sdk.NewCoin(types.Denom, initial.QuoRaw(2))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, delegatorI, expRewards).Return(nil)

	_, err = k.WithdrawDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI)
	require.Nil(t, err)

	// historical count should still be 2 (added one record, cleared one)
	require.Equal(t, 2, getRepHistoricalReferenceCount(k, sdkCtx))

	// withdraw commission (the bank keeper should be called with the right amount of tokens to transfer)
	expCommission := sdk.Coins{sdk.NewCoin(types.Denom, initial.QuoRaw(2))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, reporterAccI, expCommission).Return(nil)
	_, err = k.WithdrawReporterCommission(sdkCtx, reporterAccI.Bytes())
	require.Nil(t, err)
}

func TestCalculateRewardsAfterManySlashesInSameBlock(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.IsZero())

	// start out block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// allocate some rewards
	initial := math.LegacyNewDecFromInt(sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction))
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: initial}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// slash the reporter by 50% (simulated with manual calls; we assume the reporter is bonded)
	reporter, burned := SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)
	// slash the reporter by 50% again
	reporter, burned = SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	// increase block height
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards should be half the tokens
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: initial}}, rewards)

	// commission should be the other half
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: initial}}, repCommission.Commission)
}

func TestCalculateRewardsMultiDelegatorMultiSlash(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	initial := math.LegacyNewDecFromInt(sdk.TokensFromConsensusPower(30, sdk.DefaultPowerReduction))
	tokens := sdk.DecCoins{{Denom: types.Denom, Amount: initial}}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// slash the reporter
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)
	// slash the reporter by 50% (simulated with manual calls; we assume the reporter is bonded)
	reporter, burned := SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// second delegation
	delegatorII := sdk.AccAddress([]byte("delegator2"))
	delegationII := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.BeforeDelegationCreated(sdkCtx, reporter)
	require.NoError(t, err)
	err = k.Delegators.Set(sdkCtx, delegatorII, delegationII)
	require.NoError(t, err)
	// call necessary hooks to update a delegation
	// end period
	err = k.AfterDelegationModified(sdkCtx, delegatorII, reporterAccI.Bytes(), delegationII.Amount)
	require.NoError(t, err)
	// update reporter with new total tokens from the second delegation
	reporter.TotalTokens = reporter.TotalTokens.Add(delegationII.Amount)
	err = k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// slash the reporter again
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)
	reporter, burned = SlashReporter(sdkCtx, reporterAccI.Bytes(), sdkCtx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1), reporter, k)
	require.True(t, burned.IsPositive(), "expected positive slashed tokens, got: %s", burned)

	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 3)

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards for del1 should be 2/3 initial (half initial first period, 1/6 initial second period)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: initial.QuoInt64(2).Add(initial.QuoInt64(6))}}, rewards)

	// calculate delegation rewards for del2
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII, delegationII, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be initial / 3
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: initial.QuoInt64(3)}}, rewards)

	// commission should be equal to initial (twice 50% commission, unaffected by slashing)
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: initial}}, repCommission.Commission)
}

func TestCalculateRewardsMultiDelegatorMultWithdraw(t *testing.T) {
	k, _, bk, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	initial := int64(20)
	tokens := sdk.DecCoins{sdk.NewDecCoin(types.Denom, math.NewInt(initial))}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// historical count should be 2 (reporter init, delegation init)
	require.Equal(t, 2, getRepHistoricalReferenceCount(k, sdkCtx))

	// second delegation
	delegatorII := sdk.AccAddress([]byte("delegator2"))
	delegationII := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.BeforeDelegationCreated(sdkCtx, reporter)
	require.NoError(t, err)
	err = k.Delegators.Set(sdkCtx, delegatorII, delegationII)
	require.NoError(t, err)
	// call necessary hooks to update a delegation
	// end period
	err = k.AfterDelegationModified(sdkCtx, delegatorII, reporterAccI.Bytes(), delegationII.Amount)
	require.NoError(t, err)
	// update reporter with new total tokens from the second delegation
	reporter.TotalTokens = reporter.TotalTokens.Add(delegationII.Amount)
	err = k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// historical count should be 3 (second delegation init)
	require.Equal(t, 3, getRepHistoricalReferenceCount(k, sdkCtx))

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// first delegator withdraws
	expRewards := sdk.Coins{sdk.NewCoin(types.Denom, math.NewInt(initial*3/4))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, delegatorI, expRewards).Return(nil)
	_, err = k.WithdrawDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI)
	require.Nil(t, err)

	// second delegator withdraws
	expRewards = sdk.Coins{sdk.NewCoin(types.Denom, math.NewInt(initial*1/4))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, delegatorII, expRewards).Return(nil)
	_, err = k.WithdrawDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII)
	require.Nil(t, err)

	// historical count should be 3 (reporter init + two delegations)
	require.Equal(t, 3, getRepHistoricalReferenceCount(k, sdkCtx))

	// reporter withdraws commission
	expCommission := sdk.Coins{sdk.NewCoin(types.Denom, math.NewInt(initial))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, reporterAccI, expCommission).Return(nil)
	_, err = k.WithdrawReporterCommission(sdkCtx, reporterAccI.Bytes())
	require.Nil(t, err)

	// end period
	endingPeriod, _ := k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err := k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards for del1 should be zero
	require.True(t, rewards.IsZero())

	// calculate delegation rewards for del2
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII, delegationII, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be zero
	require.True(t, rewards.IsZero())

	// commission should be zero
	repCommission, err := k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.True(t, repCommission.Commission.IsZero())

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(ctx, reporterAccI.Bytes(), tokens))

	// first delegator withdraws again
	expRewards = sdk.Coins{sdk.NewCoin(types.Denom, math.NewInt(initial*1/4))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, delegatorI, expRewards).Return(nil)
	_, err = k.WithdrawDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI)
	require.Nil(t, err)

	// end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards for del1 should be zero
	require.True(t, rewards.IsZero())

	// calculate delegation rewards for del2
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII, delegationII, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 4)}}, rewards)

	// commission should be half initial
	repCommission, err = k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 2)}}, repCommission.Commission)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// withdraw commission
	expCommission = sdk.Coins{sdk.NewCoin(types.Denom, math.NewInt(initial))}
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, types.ModuleName, reporterAccI, expCommission).Return(nil)
	_, err = k.WithdrawReporterCommission(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)

	// end period
	endingPeriod, _ = k.IncrementReporterPeriod(sdkCtx, reporter)

	// calculate delegation rewards for del1
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI, delegationI, endingPeriod)
	require.NoError(t, err)

	// rewards for del1 should be 1/4 initial
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 4)}}, rewards)

	// calculate delegation rewards for del2
	rewards, err = k.CalculateDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorII, delegationII, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be 1/2 initial
	require.Equal(t, sdk.DecCoins{{Denom: types.Denom, Amount: math.LegacyNewDec(initial / 2)}}, rewards)

	// commission should be zero
	repCommission, err = k.ReportersAccumulatedCommission.Get(sdkCtx, reporterAccI.Bytes())
	require.NoError(t, err)
	require.True(t, repCommission.Commission.IsZero())
}

func Test100PercentCommissionReward(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// create reporter with 50% commission
	reporterAccI := sdk.AccAddress([]byte("reporter1"))
	commission := types.NewCommissionWithTime(math.LegacyNewDecWithPrec(10, 1), math.LegacyNewDecWithPrec(10, 1),
		math.LegacyNewDec(0), time.Time{})
	reporter := types.NewOracleReporter(reporterAccI.String(), math.NewInt(100), &commission)
	err := k.Reporters.Set(sdkCtx, reporterAccI, reporter)
	require.NoError(t, err)

	// self delegation
	delegatorI := reporterAccI
	delegationI := types.Delegation{Reporter: reporterAccI.Bytes(), Amount: math.NewInt(100)}
	err = k.Delegators.Set(sdkCtx, delegatorI, delegationI)
	require.NoError(t, err)

	// run the necessary hooks manually (given that we are not running an actual staking module)
	err = CallCreateReporterHooks(sdkCtx, k, delegatorI, reporter, delegationI.Amount)
	require.NoError(t, err)

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	initial := int64(20)
	tokens := sdk.DecCoins{sdk.NewDecCoin(types.Denom, math.NewInt(initial))}
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))

	// next block
	sdkCtx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 1)

	// allocate some more rewards
	require.NoError(t, k.AllocateTokensToReporter(sdkCtx, reporterAccI.Bytes(), tokens))
	rewards, err := k.WithdrawDelegationRewards(sdkCtx, reporterAccI.Bytes(), delegatorI)
	require.NoError(t, err)

	zeroRewards := sdk.Coins{sdk.NewCoin(types.Denom, math.ZeroInt())}
	require.True(t, rewards.Equal(zeroRewards))

	events := sdkCtx.EventManager().Events()
	lastEvent := events[len(events)-1]

	var hasValue bool
	for _, attr := range lastEvent.Attributes {
		if attr.Key == "amount" && attr.Value == "0loya" {
			hasValue = true
		}
	}
	require.True(t, hasValue)
}

// ******************** helpers *********************

// SlashReporter copies what x/staking Slash does. It should be used for testing only.
// And it must be updated whenever the original function is updated.
// The passed reporter will get its tokens updated.
func SlashReporter(
	ctx sdk.Context,
	reporterVal sdk.ValAddress,
	infractionHeight int64,
	slashFactor math.LegacyDec,
	reporter types.OracleReporter,
	k keeper.Keeper,
) (types.OracleReporter, math.Int) {
	if slashFactor.IsNegative() {
		panic(fmt.Errorf("attempted to slash with a negative slash factor: %v", slashFactor))
	}

	// we simplify this part, as we won't be able to test redelegations or
	// unbonding delegations
	if infractionHeight != ctx.BlockHeight() {
		// if a new test lands here we might need to update this function to handle redelegations and unbonding
		// or just make it an integration test.
		panic("we can't test any other case here")
	}

	slashAmountDec := math.LegacyNewDecFromInt(reporter.TotalTokens).Mul(math.LegacyNewDecWithPrec(5, 1))
	slashAmount := slashAmountDec.TruncateInt()

	// cannot decrease balance below zero
	tokensToBurn := math.MinInt(slashAmount, reporter.TotalTokens)
	tokensToBurn = math.MaxInt(tokensToBurn, math.ZeroInt()) // defensive.

	// we need to calculate the *effective* slash fraction for distribution
	if reporter.TotalTokens.IsPositive() {
		effectiveFraction := math.LegacyNewDecFromInt(tokensToBurn).QuoRoundUp(math.LegacyNewDecFromInt(reporter.TotalTokens))
		// possible if power has changed
		if effectiveFraction.GT(math.LegacyOneDec()) {
			effectiveFraction = math.LegacyOneDec()
		}
		// call the before-slashed hook
		err := k.BeforeReporterDisputed(ctx, reporterVal, effectiveFraction)
		if err != nil {
			panic(err)
		}
	}
	// Deduct from reporter's bonded tokens and update the reporter.
	// Burn the slashed tokens from the pool account and decrease the total supply.
	reporter.TotalTokens = reporter.TotalTokens.Sub(tokensToBurn)
	err := k.Reporters.Set(ctx, sdk.AccAddress(reporterVal), reporter)
	if err != nil {
		panic(err)
	}

	return reporter, tokensToBurn
}

func getRepHistoricalReferenceCount(k keeper.Keeper, ctx sdk.Context) int {
	count := 0
	err := k.ReporterHistoricalRewards.Walk(
		ctx, nil, func(key collections.Pair[[]byte, uint64], rewards types.ReporterHistoricalRewards) (stop bool, err error) {
			count += int(rewards.ReferenceCount)
			return false, nil
		},
	)
	if err != nil {
		panic(err)
	}

	return count
}

func CallCreateReporterHooks(sdkCtx sdk.Context, k keeper.Keeper, delegator sdk.AccAddress, reporter types.OracleReporter, stake math.Int) error {
	// hooks
	err := k.AfterReporterCreated(sdkCtx, reporter)
	if err != nil {
		return err
	}

	_, err = k.IncrementReporterPeriod(sdkCtx, reporter)
	if err != nil {
		return err
	}

	return k.AfterDelegationModified(sdkCtx, delegator, reporter.GetReporter(), stake)
}
