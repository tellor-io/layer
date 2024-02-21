package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestDelegateReporter(t *testing.T) {
	k, sk, _, ms, ctx := setupMsgServer(t)

	delAddr := sdk.AccAddress([]byte("delegator"))
	repAddr := sdk.AccAddress([]byte("reporter"))
	validatorI := sdk.ValAddress([]byte("validator1"))
	validatorII := sdk.ValAddress([]byte("validator2"))

	msg := types.MsgDelegateReporter{
		Delegator: delAddr.String(),
		Reporter:  repAddr.String(),
		Amount:    math.NewInt(100),
		TokenOrigins: []*types.TokenOrigin{
			{
				ValidatorAddress: validatorI.String(),
				Amount:           math.NewInt(50),
			},
			{
				ValidatorAddress: validatorII.String(),
				Amount:           math.NewInt(50),
			},
		},
	}

	// add the reporter to the keeper
	reporter := types.OracleReporter{Reporter: repAddr.String(), TotalTokens: math.NewInt(200)}
	err := k.Reporters.Set(ctx, repAddr, reporter)
	require.NoError(t, err)
	// call distr hooks
	err = k.AfterReporterCreated(ctx, reporter)
	require.NoError(t, err)

	err = k.BeforeDelegationCreated(ctx, reporter)
	require.NoError(t, err)

	// set up mock for ValidateAmount method of Keeper
	delegationI := stakingtypes.Delegation{Shares: math.LegacyNewDec(50), DelegatorAddress: delAddr.String(), ValidatorAddress: validatorI.String()}
	valI := stakingtypes.Validator{OperatorAddress: validatorI.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}
	delegationII := stakingtypes.Delegation{Shares: math.LegacyNewDec(50), DelegatorAddress: delAddr.String(), ValidatorAddress: validatorII.String()}
	valII := stakingtypes.Validator{OperatorAddress: validatorII.String(), Tokens: math.NewInt(100), DelegatorShares: math.LegacyNewDec(50)}

	// mock the staking keeper
	sk.On("GetValidator", mock.Anything, validatorI).Return(valI, nil)
	sk.On("GetValidator", mock.Anything, validatorII).Return(valII, nil)
	sk.On("Delegation", mock.Anything, delAddr, validatorI).Return(delegationI, nil)
	sk.On("Delegation", mock.Anything, delAddr, validatorII).Return(delegationII, nil)

	// Call the DelegateReporter function
	res, err := ms.DelegateReporter(ctx, &msg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Check if the delegation was added correctly
	delegation, err := k.Delegators.Get(ctx, delAddr)
	require.NoError(t, err)
	require.Equal(t, repAddr.String(), delegation.Reporter)
}
