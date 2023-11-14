package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
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
	require.NotNil(t, getSpec)
	//fmt.Println(getSpec)

	// try to get spec with bad types.QueryGetDataSpecRequest
	getSpec, err = k.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "badQueryType"})
	require.Nil(t, err)
	//fmt.Println(getSpec)
	//fmt.Println(err)
	require.NotNil(t, getSpec)

}
