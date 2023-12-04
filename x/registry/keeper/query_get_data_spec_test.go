package keeper_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	types "github.com/tellor-io/layer/x/registry/types"
)

func TestQueryGetDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// register a spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// get DataSpec
	getSpec, err := k.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "queryType1"})
	require.NoError(t, err)
	require.Equal(t, getSpec.Spec.DocumentHash, "hash1")
	require.Equal(t, getSpec.Spec.ValueType, "uint256")

	// get unregistered spec
	getSpec, err = k.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "badQueryType"})
	require.Nil(t, err)
	require.NotNil(t, getSpec)
	require.Equal(t, getSpec.Spec.DocumentHash, "")
	require.Equal(t, getSpec.Spec.ValueType, "")

	// get empty spec
	getSpec, err = k.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{})
	require.Nil(t, err)
	require.NotNil(t, getSpec)
	require.Equal(t, getSpec.Spec.DocumentHash, "")
	require.Equal(t, getSpec.Spec.ValueType, "")

}

func TestQueryGetDataSpecSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// check Spec() return for unregistered data spec
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	specReturn := k.Spec(unwrappedCtx, "queryType1")
	require.Nil(t, specReturn)

	// register a spec and check Spec() returns correct bytes
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)
	unwrappedCtx = sdk.UnwrapSDKContext(ctx)
	specReturn = k.Spec(unwrappedCtx, "queryType1")
	fmt.Println("specReturn2: ", specReturn)
	data, err := proto.Marshal(&spec1)
	require.Nil(t, err)
	require.Equal(t, specReturn, data)

}
