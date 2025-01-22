package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestDecodeQuerydata(t *testing.T) {
	// register data spec
	ms, ctx, k := setupMsgServer(t)
	msgres, err := ms.RegisterSpec(ctx, &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: testQueryType,
		Spec: types.DataSpec{
			AggregationMethod: "weighted-median",
			ResponseValueType: "uint256",
			AbiComponents: []*types.ABIComponent{
				{Name: "test", FieldType: "string"},
			},
			QueryType: testQueryType,
		},
	})
	require.NoError(t, err)
	require.Equal(t, msgres, &types.MsgRegisterSpecResponse{})
	// generate query data
	querier := keeper.NewQuerier(k)
	qData, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000d74657374517565727954797065000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000")
	res, err := querier.DecodeQuerydata(ctx, &types.QueryDecodeQuerydataRequest{QueryData: qData})
	require.NoError(t, err)
	require.Equal(t, res, &types.QueryDecodeQuerydataResponse{Spec: `testQueryType: ["test"]`})
}
