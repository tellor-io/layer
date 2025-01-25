package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (s *IntegrationTestSuite) TestRegistryKeeper() {
	ms := keeper.NewMsgServerImpl(s.Setup.Registrykeeper)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	queryType := "testQueryType"
	registrar := sample.AccAddress()
	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		ReportBlockWindow: 2,
		QueryType:         queryType,
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
		QueryType: queryType,
		Spec:      spec,
	}
	_, err := ms.RegisterSpec(s.Setup.Ctx, registerSpecInput)
	s.NoError(err)
	// Update spec
	spec.ResponseValueType = "uint128"
	updateSpecInput := &types.MsgUpdateDataSpec{
		Authority: authority,
		QueryType: queryType,
		Spec:      spec,
	}
	_, err = ms.UpdateDataSpec(s.Setup.Ctx, updateSpecInput)
	s.NoError(err)

	// Check if spec is updated
	getSpec, err := s.Setup.Registrykeeper.GetSpec(s.Setup.Ctx, queryType)
	s.NoError(err)
	s.Equal("uint128", getSpec.ResponseValueType)

	// Update spec with invalid authority
	authority = "invalidAuthority"
	// update spec
	spec.ResponseValueType = "int256"
	updateSpecInput = &types.MsgUpdateDataSpec{
		Authority: authority,
		QueryType: queryType,
		Spec:      spec,
	}
	_, err = ms.UpdateDataSpec(s.Setup.Ctx, updateSpecInput)
	s.ErrorContains(err, "invalidAuthority")
}
