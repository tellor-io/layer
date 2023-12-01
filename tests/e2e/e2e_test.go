package e2e_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	disputeKeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputeType "github.com/tellor-io/layer/x/dispute/types"
	oracleKeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracleType "github.com/tellor-io/layer/x/oracle/types"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
	registryTypes "github.com/tellor-io/layer/x/registry/types"
)

func (s *IntegrationTestSuite) oracleKeeper() (queryClient oracleType.QueryClient, msgServer oracleType.MsgServer) {
	oracleType.RegisterQueryServer(s.queryHelper, s.oraclekeeper)
	oracleType.RegisterInterfaces(s.interfaceRegistry)
	queryClient = oracleType.NewQueryClient(s.queryHelper)
	msgServer = oracleKeeper.NewMsgServerImpl(s.oraclekeeper)
	return
}

func (s *IntegrationTestSuite) disputeKeeper() (queryClient disputeType.QueryClient, msgServer disputeType.MsgServer) {
	disputeType.RegisterQueryServer(s.queryHelper, s.disputekeeper)
	disputeType.RegisterInterfaces(s.interfaceRegistry)
	queryClient = disputeType.NewQueryClient(s.queryHelper)
	msgServer = disputeKeeper.NewMsgServerImpl(s.disputekeeper)
	return
}

func (s *IntegrationTestSuite) registryKeeper() (queryClient registryTypes.QueryClient, msgServer registryTypes.MsgServer) {
	registryTypes.RegisterQueryServer(s.queryHelper, s.registrykeeper)
	registryTypes.RegisterInterfaces(s.interfaceRegistry)
	queryClient = registryTypes.NewQueryClient(s.queryHelper)
	msgServer = registryKeeper.NewMsgServerImpl(s.registrykeeper)
	return
}

func (s *IntegrationTestSuite) TestUseAllModules() {
	k, msgServerOracle := s.oracleKeeper()
	require.NotNil(s.T(), msgServerOracle)
	require.NotNil(s.T(), k)
	k2, msgServerDispute := s.disputeKeeper()
	require.NotNil(s.T(), msgServerDispute)
	require.NotNil(s.T(), k2)
	k3, msgServerRegistry := s.registryKeeper()
	require.NotNil(s.T(), msgServerRegistry)
	require.NotNil(s.T(), k3)

	// register a spec
	spec1 := registryTypes.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &registryTypes.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := msgServerRegistry.RegisterSpec(s.ctx, specInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), registerSpecResult)

	// register query for spec1
	queryInput := &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err := msgServerRegistry.RegisterQuery(s.ctx, queryInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), registerQueryResult)
	//get query data for query1
	unwrappedCtx := sdk.UnwrapSDKContext(s.ctx)
	queryData, err := k3.GetQueryData(unwrappedCtx, &types.QueryGetQueryDataRequest{QueryId: registerQueryResult.QueryId})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), queryData)

	//submit a value of that query
	submitResult, err := msgServerOracle.SubmitValue(s.ctx, &oracleType.MsgSubmitValue{Creator: "creator1", QueryData: queryData.QueryData, Value: "1"})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), submitResult)

}
