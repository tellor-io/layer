package integration_test

import (
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (s *IntegrationTestSuite) TestRegistryKeeper() {
	ms := keeper.NewMsgServerImpl(s.registrykeeper)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	queryType := "testQueryType"
	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
	}

	// Register spec
	registerSpecInput := &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: queryType,
		Spec:      spec,
	}
	_, err := ms.RegisterSpec(s.ctx, registerSpecInput)
	s.NoError(err)
	// Update spec
	spec.ResponseValueType = "uint128"
	updateSpecInput := &types.MsgUpdateDataSpec{
		Authority: authority,
		QueryType: queryType,
		Spec:      spec,
	}
	_, err = ms.UpdateDataSpec(s.ctx, updateSpecInput)
	s.NoError(err)

	// Check if spec is updated
	getSpec, err := s.registrykeeper.GetSpec(s.ctx, queryType)
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
	_, err = ms.UpdateDataSpec(s.ctx, updateSpecInput)
	s.ErrorContains(err, "invalidAuthority")
}
