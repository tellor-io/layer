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
	spec1 := types.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	// try to register spec that already exists
	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.ErrorContains(t, err, "data spec previously registered")
	require.Nil(t, registerSpecResult)

	// register invalid value type
	spec2 := types.DataSpec{DocumentHash: "hash1", ValueType: "fakeValueType"}
	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType2",
		Spec:      spec2,
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.ErrorContains(t, err, "value type not supported")
	require.Nil(t, registerSpecResult)

	// register each supported type
	type1, type2, type3, type4 := "string", "bool", "address", "bytes"
	type5, type6, type7, type8, type9, type10 := "int8", "int16", "int32", "int64", "int128", "int256"
	type11, type12, type13, type14, type15 := "uint8", "uint16", "uint32", "uint64", "uint128" //uint256 already done

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType3",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type1},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType4",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type2},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType5",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type3},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType6",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type4},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType7",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type5},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType8",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type6},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType9",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type7},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType10",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type8},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType11",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type9},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType12",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type10},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType13",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type11},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType14",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type12},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType15",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type13},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType16",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type14},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

	specInput = &types.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType17",
		Spec:      types.DataSpec{DocumentHash: "hash1", ValueType: type15},
	}
	registerSpecResult, err = ms.RegisterSpec(ctx, specInput)
	require.NoError(t, err)
	require.NotNil(t, registerSpecResult)

}
