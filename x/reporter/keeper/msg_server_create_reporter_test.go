package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestCreateReporter(t *testing.T) {
	k, sk, ms, ctx := setupMsgServer(t)

	// setup delegator and validators
	reporterAddr := sdk.AccAddress([]byte("reporter"))
	reporterBech32 := reporterAddr.String()
	amount := math.NewInt(100)
	commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(3, 1),
		math.LegacyNewDecWithPrec(1, 1), time.Time{})

	validatorI := sdk.ValAddress([]byte("validator1"))
	validatorII := sdk.ValAddress([]byte("validator2"))
	validatorIII := sdk.ValAddress([]byte("validator3"))
	delegationI := stakingtypes.Delegation{Shares: math.LegacyNewDec(50), DelegatorAddress: reporterBech32, ValidatorAddress: validatorI.String()}
	valI := stakingtypes.Validator{OperatorAddress: validatorI.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	delegationII := stakingtypes.Delegation{Shares: math.LegacyNewDec(30), DelegatorAddress: reporterBech32, ValidatorAddress: validatorII.String()}
	valII := stakingtypes.Validator{OperatorAddress: validatorII.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	delegationIII := stakingtypes.Delegation{Shares: math.LegacyNewDec(20), DelegatorAddress: reporterBech32, ValidatorAddress: validatorIII.String()}
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
	// mock the staking keeper
	sk.On("GetValidator", mock.Anything, validatorI).Return(valI, nil)
	sk.On("GetValidator", mock.Anything, validatorII).Return(valII, nil)
	sk.On("GetValidator", mock.Anything, validatorIII).Return(valIII, nil)
	sk.On("Delegation", mock.Anything, reporterAddr, validatorI).Return(delegationI, nil)
	sk.On("Delegation", mock.Anything, reporterAddr, validatorII).Return(delegationII, nil)
	sk.On("Delegation", mock.Anything, reporterAddr, validatorIII).Return(delegationIII, nil)

	msg := types.NewMsgCreateReporter(reporterBech32, amount, originAmounts, &commission)
	res, err := ms.CreateReporter(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// verify that the reporter exists in the keeper
	exists, err := k.Reporters.Has(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, exists)

	// verify that the delegation exists in the keeper
	exists, err = k.Delegators.Has(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, exists)

	// verify that the delegation exists in the keeper
	delegatorExists, err := k.Delegators.Has(ctx, reporterAddr)
	require.NoError(t, err)
	require.True(t, delegatorExists)

	// verify that the commission is set correctly
	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	require.NoError(t, err)
	require.Equal(t, commission.Rate, reporter.Commission.Rate)
	require.Equal(t, commission.MaxRate, reporter.Commission.MaxRate)
	require.Equal(t, commission.MaxChangeRate, reporter.Commission.MaxChangeRate)
}
