package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestRegisterSpec(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	// register a spec
	spec1 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median", QueryType: testQueryType}
	specInput := &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: testQueryType,
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// try to register spec that already exists
	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: testQueryType,
		Spec:      spec1,
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.ErrorContains(t, err, "data spec previously registered")
	require.Nil(t, registerSpecResult)

	// register invalid value type
	spec2 := types.DataSpec{DocumentHash: "hash1", ResponseValueType: "fakeValueType", AggregationMethod: "weighted-median", QueryType: "querytype2"}
	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype2",
		Spec:      spec2,
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.ErrorContains(t, err, "value type not supported")
	require.Nil(t, registerSpecResult)

	// register each supported type
	type1, type2, type3, type4 := "string", "bool", "address", "bytes"
	type5, type6, type7, type8, type9, type10 := "int8", "int16", "int32", "int64", "int128", "int256"
	type11, type12, type13, type14, type15 := "uint8", "uint16", "uint32", "uint64", "uint128"

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype3",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type1, AggregationMethod: "weighted-median", QueryType: "querytype3"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype4",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type2, AggregationMethod: "weighted-median", QueryType: "querytype4"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype5",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type3, AggregationMethod: "weighted-median", QueryType: "querytype5"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype6",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type4, AggregationMethod: "weighted-median", QueryType: "querytype6"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype7",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type5, AggregationMethod: "weighted-median", QueryType: "querytype7"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype8",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type6, AggregationMethod: "weighted-median", QueryType: "querytype8"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype9",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type7, AggregationMethod: "weighted-median", QueryType: "querytype9"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype10",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type8, AggregationMethod: "weighted-median", QueryType: "querytype10"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype11",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type9, AggregationMethod: "weighted-median", QueryType: "querytype11"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype12",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type10, AggregationMethod: "weighted-median", QueryType: "querytype12"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype13",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type11, AggregationMethod: "weighted-median", QueryType: "querytype13"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype14",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type12, AggregationMethod: "weighted-median", QueryType: "querytype14"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype15",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type13, AggregationMethod: "weighted-median", QueryType: "querytype15"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "querytype16",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type14, AggregationMethod: "weighted-median", QueryType: "queryType16"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: "queryType17",
		Spec:      types.DataSpec{DocumentHash: "hash1", ResponseValueType: type15, AggregationMethod: "weighted-median", QueryType: "querytype17"},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)
}
