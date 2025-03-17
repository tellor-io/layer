package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/registry/keeper"
	registry "github.com/tellor-io/layer/x/registry/module"
	types "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestQueryGetDataSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	querier := keeper.NewQuerier(k)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	registrar := sample.AccAddress()

	// register a spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median", QueryType: "querytype1", Registrar: registrar, AbiComponents: []*types.ABIComponent{
		{
			Name:      "field",
			FieldType: "uint256",
		},
	}}
	specInput := &types.MsgRegisterSpec{
		Registrar: registrar,
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
	_, err = querier.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{QueryType: "badQueryType"})
	require.ErrorContains(t, err, "data spec not registered")

	// get empty spec
	_, err = querier.GetDataSpec(ctx, &types.QueryGetDataSpecRequest{})
	require.ErrorContains(t, err, "query type cannot be empty")
}

// genesis data specs and 1 additional spec
func TestQueryGetAllDataSpecs(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	querier := keeper.NewQuerier(k)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	registrar := sample.AccAddress()
	registry.InitGenesis(sdk.UnwrapSDKContext(ctx), k, types.GenesisState{
		Params:   types.DefaultParams(),
		Dataspec: types.GenesisDataSpec(), // spotprice and trbbridge
	})

	// register an additional spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median", QueryType: "querytype1", Registrar: registrar, AbiComponents: []*types.ABIComponent{
		{
			Name:      "field",
			FieldType: "uint256",
		},
	}}
	specInput := &types.MsgRegisterSpec{
		Registrar: registrar,
		QueryType: "querytype1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// get all specs
	getAllSpecs, err := querier.GetAllDataSpecs(ctx, &types.QueryGetAllDataSpecsRequest{})
	fmt.Println("getAllSpecs", getAllSpecs)
	require.NoError(t, err)
	require.NotNil(t, getAllSpecs)
	require.Equal(t, len(getAllSpecs.Specs), 3)
	for _, spec := range getAllSpecs.Specs {
		fmt.Println("spec", spec)
		require.Contains(t, []string{"spotprice", "trbbridge", "querytype1"}, spec.QueryType)
	}
}
