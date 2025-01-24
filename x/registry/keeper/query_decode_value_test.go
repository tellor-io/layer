package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestDecodeValueQuery(t *testing.T) {
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
	res, err := querier.DecodeValue(ctx, &types.QueryDecodeValueRequest{QueryType: testQueryType, Value: "0x00000000000000000000000000000000000000000000090ea01c800d96350000"})
	require.NoError(t, err)
	require.Equal(t, res, &types.QueryDecodeValueResponse{DecodedValue: "[42771090000000000000000]"}) // 42771_090_000_000_000_000_000 == 42,771.09
}
