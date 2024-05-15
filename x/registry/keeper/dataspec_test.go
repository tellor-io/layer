package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestQueryGetDataSpecSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// check Spec() return for unregistered data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.GetSpec(unwrappedCtx, "queryType1")
	require.Error(t, err)
	require.Equal(t, specReturn, types.DataSpec{})

	// register a spec and check Spec() returns correct bytes
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median", Registrar: "creator1"}
	specInput := &types.MsgRegisterSpec{
		Registrar: spec1.Registrar,
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.Equal(t, registerSpecResult, &types.MsgRegisterSpecResponse{})

	specReturn, err = k.GetSpec(unwrappedCtx, "queryType1")
	fmt.Println("specReturn2: ", specReturn)
	require.Nil(t, err)
	require.Equal(t, specReturn, spec1)

}
func TestSetDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// Define test data
	queryType := "queryType1"
	dataSpec := types.DataSpec{
		DocumentHash:      "hash1",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         "creator1",
	}

	// Call the function
	err := k.SetDataSpec(sdk.UnwrapSDKContext(ctx), queryType, dataSpec)
	require.NoError(t, err)

	// Retrieve the data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.GetSpec(unwrappedCtx, queryType)
	require.NoError(t, err)
	require.Equal(t, specReturn, dataSpec)
}

func TestHasDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// Define test data
	queryType := "queryType1"
	dataSpec := types.DataSpec{
		DocumentHash:      "hash1",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		Registrar:         "creator1",
	}

	// Call the function
	err := k.SetDataSpec(sdk.UnwrapSDKContext(ctx), queryType, dataSpec)
	require.NoError(t, err)

	// Retrieve the data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn, err := k.HasSpec(unwrappedCtx, queryType)
	require.NoError(t, err)
	require.Equal(t, specReturn, true)
}
