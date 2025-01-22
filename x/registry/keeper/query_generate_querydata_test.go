package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestGenerateQueryData(t *testing.T) {
	// register data spec
	ms, ctx, k := setupMsgServer(t)
	msgres, err := ms.RegisterSpec(ctx, &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: testQueryType,
		Spec: types.DataSpec{
			ResponseValueType: "uint256",
			AbiComponents: []*types.ABIComponent{
				{Name: "test", FieldType: "string"},
			},
			AggregationMethod: "weighted-median",
			QueryType:         testQueryType,
		},
	})
	require.NoError(t, err)
	require.Equal(t, msgres, &types.MsgRegisterSpecResponse{})
	// generate query data
	querier := keeper.NewQuerier(k)
	queryres, err := querier.GenerateQuerydata(ctx, &types.QueryGenerateQuerydataRequest{
		Querytype:  testQueryType,
		Parameters: `["test"]`,
	})
	require.NoError(t, err)
	expectedquerydata, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000d74657374717565727974797065000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000")
	require.Equal(t, queryres, &types.QueryGenerateQuerydataResponse{QueryData: expectedquerydata})
}
