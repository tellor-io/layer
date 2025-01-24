package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestUpdateDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		QueryType:         testQueryType,
	}

	// Register spec
	registerSpecInput := &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: testQueryType,
		Spec:      spec,
	}
	_, err := ms.RegisterSpec(ctx, registerSpecInput)
	require.NoError(t, err)
	// Update spec
	spec.ResponseValueType = "uint128"
	updateSpecInput := &types.MsgUpdateDataSpec{
		Authority: authority,
		QueryType: testQueryType,
		Spec:      spec,
	}
	_, err = ms.UpdateDataSpec(ctx, updateSpecInput)
	require.NoError(t, err)

	// Check if spec is updated
	getSpec, err := k.GetSpec(sdk.UnwrapSDKContext(ctx), testQueryType)
	require.NoError(t, err)
	require.Equal(t, "uint128", getSpec.ResponseValueType)

	// Update spec with invalid authority
	authority = "invalidAuthority"
	// update spec
	spec.ResponseValueType = "int256"
	updateSpecInput = &types.MsgUpdateDataSpec{
		Authority: authority,
		QueryType: testQueryType,
		Spec:      spec,
	}
	_, err = ms.UpdateDataSpec(ctx, updateSpecInput)
	require.ErrorContains(t, err, "invalidAuthority")
}
