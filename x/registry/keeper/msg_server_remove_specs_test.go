package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestRemoveDataSpecs(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	registrar := sample.AccAddress()

	badSenderMsg := &types.MsgRemoveDataSpecs{
		Authority:     registrar,
		DataSpecTypes: make([]string, 0),
	}

	_, err := ms.RemoveDataSpecs(ctx, badSenderMsg)
	require.ErrorContains(t, err, "invalid authority")

	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		QueryType:         "test_spec",
		Registrar:         registrar,
		AbiComponents: []*types.ABIComponent{
			{
				Name:      "field",
				FieldType: "uint256",
			},
		},
	}

	// Register spec
	registerSpecInput := &types.MsgRegisterSpec{
		Registrar: registrar,
		QueryType: "test_spec",
		Spec:      spec,
	}
	_, err = ms.RegisterSpec(ctx, registerSpecInput)
	require.NoError(t, err)

	spec2 := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		QueryType:         "test_spec2",
		Registrar:         registrar,
		AbiComponents: []*types.ABIComponent{
			{
				Name:      "field",
				FieldType: "uint256",
			},
		},
	}

	// Register spec
	registerSpecInput2 := &types.MsgRegisterSpec{
		Registrar: registrar,
		QueryType: spec2.QueryType,
		Spec:      spec2,
	}
	_, err = ms.RegisterSpec(ctx, registerSpecInput2)
	require.NoError(t, err)

	removeSpecsRequest := &types.MsgRemoveDataSpecs{
		Authority:     k.GetAuthority(),
		DataSpecTypes: []string{"test_spec", "test_spec2"},
	}

	_, err = ms.RemoveDataSpecs(ctx, removeSpecsRequest)
	require.NoError(t, err)

}
