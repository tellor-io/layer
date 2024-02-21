package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestUndelegateReporter(t *testing.T) {
	k, _, _, ms, ctx := setupMsgServer(t)

	delAddr := sdk.AccAddress([]byte("delegator"))
	repAddr := sdk.AccAddress([]byte("reporter"))
	valAddr := sdk.ValAddress([]byte("validator"))

	reporter := types.OracleReporter{
		Reporter:    repAddr.String(),
		TotalTokens: math.NewInt(50),
	}
	delegation := types.Delegation{
		Reporter: repAddr.String(),
		Amount:   math.NewInt(50),
	}
	err := k.Delegators.Set(ctx, delAddr, delegation)
	require.NoError(t, err)
	tokenOrigin := types.TokenOrigin{
		ValidatorAddress: valAddr.String(),
		Amount:           math.NewInt(50),
	}
	err = k.TokenOrigin.Set(ctx, collections.Join(delAddr, valAddr), tokenOrigin)
	require.NoError(t, err)
	err = k.Reporters.Set(ctx, repAddr, reporter)
	require.NoError(t, err)
	// distr hooks
	err = k.AfterReporterCreated(ctx, reporter)
	require.NoError(t, err)
	err = k.BeforeDelegationCreated(ctx, reporter)
	require.NoError(t, err)
	err = k.AfterDelegationModified(ctx, delAddr, sdk.ValAddress(repAddr), delegation.Amount)
	require.NoError(t, err)

	// check if delegation and reporter exist
	delegation, err = k.Delegators.Get(ctx, delAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), delegation.Amount)
	reporter, err = k.Reporters.Get(ctx, repAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(50), reporter.TotalTokens)

	// call undelgate reporter
	msg := types.NewMsgUndelegateReporter(delAddr.String(), []*types.TokenOrigin{&tokenOrigin})
	res, err := ms.UndelegateReporter(ctx, msg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// check if delegation and reporter are removed
	// check if delegation and reporter exist
	delegation, err = k.Delegators.Get(ctx, delAddr)
	require.ErrorIs(t, err, collections.ErrNotFound)
	reporter, err = k.Reporters.Get(ctx, repAddr)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
