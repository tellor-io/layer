package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestUpdateOrRemoveSource(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)

	// Create a test key
	reporterAddr := sdk.AccAddress([]byte("reporter"))
	validatorAddr := sdk.ValAddress([]byte("validator"))
	key := collections.Join(reporterAddr, validatorAddr)

	// Add the token origin to the keeper
	err := k.TokenOrigin.Set(ctx, key, math.NewInt(100))
	require.NoError(t, err)

	// Call the updateOrRemoveSource function with a reduction amount of 50
	err = k.UpdateOrRemoveSource(ctx, key, math.NewInt(100), math.NewInt(50))
	require.NoError(t, err)

	// Check if the token origin was updated correctly
	updatedTokenOrigin, err := k.TokenOrigin.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), updatedTokenOrigin)

	// Call the updateOrRemoveSource function with a reduction amount of 60
	err = k.UpdateOrRemoveSource(ctx, key, updatedTokenOrigin, math.NewInt(60))
	require.NoError(t, err)

	// Check if the token origin was removed
	_, err = k.TokenOrigin.Get(ctx, key)
	require.Error(t, err)
}
func TestUpdateOrRemoveReporter(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)

	// Create a test reporter
	reporterAddr := sdk.AccAddress([]byte("reporter"))
	rep := types.OracleReporter{
		Reporter:    reporterAddr.String(),
		TotalTokens: math.NewInt(100),
	}

	// Add the reporter to the keeper
	err := k.Reporters.Set(ctx, reporterAddr, rep)
	require.NoError(t, err)

	// Call the UpdateOrRemoveReporter function with a reduction amount of 50
	err = k.UpdateOrRemoveReporter(ctx, reporterAddr, rep, math.NewInt(50))
	require.NoError(t, err)

	// Check if the reporter was updated correctly
	updatedReporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), updatedReporter.TotalTokens)

	// Call the UpdateOrRemoveReporter function with a reduction amount of 60
	err = k.UpdateOrRemoveReporter(ctx, reporterAddr, updatedReporter, math.NewInt(60))
	require.NoError(t, err)

	// Check if the reporter was removed
	_, err = k.Reporters.Get(ctx, reporterAddr)
	require.Error(t, err)
}
func TestUpdateOrRemoveDelegator(t *testing.T) {
	k, _, _, ctx := setupKeeper(t)

	// Create a test key
	delegatorAddr := sdk.AccAddress([]byte("delegator"))
	key := delegatorAddr

	// Create a test delegation
	delegation := types.Delegation{
		Reporter: delegatorAddr.String(),
		Amount:   math.NewInt(100),
	}
	reporter := types.OracleReporter{
		Reporter:    key.String(),
		TotalTokens: math.NewInt(100),
	}
	err := k.Reporters.Set(ctx, key, reporter)
	require.NoError(t, err)
	// hooks
	err = k.AfterReporterCreated(ctx, reporter)
	require.NoError(t, err)

	err = k.BeforeDelegationCreated(ctx, reporter)
	require.NoError(t, err)

	// Add the delegation to the keeper
	err = k.Delegators.Set(ctx, key, delegation)
	require.NoError(t, err)
	err = k.AfterDelegationModified(ctx, key, sdk.ValAddress(key), delegation.Amount)
	require.NoError(t, err)

	// Call the UpdateOrRemoveDelegator function with a reduction amount of 50
	err = k.UpdateOrRemoveDelegator(ctx, key, delegation, reporter, math.NewInt(50))
	require.NoError(t, err)

	// Check if the delegation was updated correctly
	updatedDelegation, err := k.Delegators.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), updatedDelegation.Amount)

	// Call the UpdateOrRemoveDelegator function with a reduction amount of 60
	err = k.UpdateOrRemoveDelegator(ctx, key, updatedDelegation, reporter, math.NewInt(60))
	require.NoError(t, err)

	// Check if the delegation was removed
	_, err = k.Delegators.Get(ctx, key)
	require.Error(t, err)
}

func TestValidateAndSetAmount(t *testing.T) {
	k, sk, _, ctx := setupKeeper(t)
	// setup
	delegator := sdk.AccAddress([]byte("delegator"))
	validatorI := sdk.ValAddress([]byte("validator1"))
	validatorII := sdk.ValAddress([]byte("validator2"))
	validatorIII := sdk.ValAddress([]byte("validator3"))
	delegationI := stakingtypes.Delegation{Shares: math.LegacyNewDec(50), DelegatorAddress: delegator.String(), ValidatorAddress: validatorI.String()}
	valI := stakingtypes.Validator{OperatorAddress: validatorI.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	delegationII := stakingtypes.Delegation{Shares: math.LegacyNewDec(30), DelegatorAddress: delegator.String(), ValidatorAddress: validatorII.String()}
	valII := stakingtypes.Validator{OperatorAddress: validatorII.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	delegationIII := stakingtypes.Delegation{Shares: math.LegacyNewDec(20), DelegatorAddress: delegator.String(), ValidatorAddress: validatorIII.String()}
	valIII := stakingtypes.Validator{OperatorAddress: validatorIII.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	originAmounts := []*types.TokenOrigin{
		{
			ValidatorAddress: validatorI.String(),
			Amount:           math.NewInt(50),
		},
		{
			ValidatorAddress: validatorII.String(),
			Amount:           math.NewInt(30),
		},
		{
			ValidatorAddress: validatorIII.String(),
			Amount:           math.NewInt(20),
		},
	}

	sk.On("GetValidator", mock.Anything, validatorI).Return(valI, nil)
	sk.On("GetValidator", mock.Anything, validatorII).Return(valII, nil)
	sk.On("GetValidator", mock.Anything, validatorIII).Return(valIII, nil)

	sk.On("Delegation", mock.Anything, delegator, validatorI).Return(delegationI, nil)
	sk.On("Delegation", mock.Anything, delegator, validatorII).Return(delegationII, nil)
	sk.On("Delegation", mock.Anything, delegator, validatorIII).Return(delegationIII, nil)

	// call ValidateAndSetAmount with amount not matching the sum of token origins
	err := k.ValidateAndSetAmount(ctx, delegator, originAmounts, math.NewInt(1000))
	require.ErrorIs(t, err, types.ErrTokenAmountMismatch)

	// call ValidateAndSetAmount with amount matching the sum of token origins
	// and no token origins set so it will treat it as a new origin
	// and check if the token origin is set correctly
	err = k.ValidateAndSetAmount(ctx, delegator, originAmounts, math.NewInt(100))
	require.NoError(t, err)
	// check if the token origin is set correctly for validatorI
	amt, err := k.TokenOrigin.Get(ctx, collections.Join(delegator, validatorI))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), amt)

	// set origin amount before calling ValidateAndSetAmount
	// and check if the token origin is updated correctly
	// should not treat it as a new origin
	k.TokenOrigin.Set(ctx, collections.Join(delegator, validatorI), math.NewInt(50))
	err = k.ValidateAndSetAmount(ctx, delegator, originAmounts, math.NewInt(100))
	require.NoError(t, err)
	amt, err = k.TokenOrigin.Get(ctx, collections.Join(delegator, validatorI))
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), amt)

	// call ValidateAndSetAmount with insufficient tokens bonded with validator
	// and check if the error is ErrInsufficientTokens
	originAmounts[0].Amount = math.NewInt(950)
	err = k.ValidateAndSetAmount(ctx, delegator, originAmounts, math.NewInt(1000))
	require.ErrorIs(t, err, types.ErrInsufficientTokens)

}
