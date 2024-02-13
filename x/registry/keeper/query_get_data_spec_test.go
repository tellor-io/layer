package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	types "github.com/tellor-io/layer/x/registry/types"
)

func TestQueryGetDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	querier := keeper.NewQuerier(k)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// register a spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median"}
	specInput := &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// get DataSpec
	getSpec, err := querier.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "queryType1"})
	require.NoError(t, err)
	require.Equal(t, getSpec.Spec.DocumentHash, "hash1")
	require.Equal(t, getSpec.Spec.ResponseValueType, "uint256")

	// get unregistered spec
	getSpec, err = querier.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "badQueryType"})
	require.ErrorContains(t, err, "data spec not registered")

	// get empty spec
	getSpec, err = querier.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{})
	require.ErrorContains(t, err, "query type cannot be empty")
}
