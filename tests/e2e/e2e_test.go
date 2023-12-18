package e2e_test

import (
	"encoding/hex"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
)

func (s *E2ETestSuite) oracleKeeper() (queryClient oracletypes.QueryClient, msgServer oracletypes.MsgServer) {
	oracletypes.RegisterQueryServer(s.queryHelper, s.oraclekeeper)
	oracletypes.RegisterInterfaces(s.interfaceRegistry)
	queryClient = oracletypes.NewQueryClient(s.queryHelper)
	msgServer = oraclekeeper.NewMsgServerImpl(s.oraclekeeper)
	return
}

func (s *E2ETestSuite) disputeKeeper() (queryClient disputetypes.QueryClient, msgServer disputetypes.MsgServer) {
	disputetypes.RegisterQueryServer(s.queryHelper, s.disputekeeper)
	disputetypes.RegisterInterfaces(s.interfaceRegistry)
	queryClient = disputetypes.NewQueryClient(s.queryHelper)
	msgServer = disputekeeper.NewMsgServerImpl(s.disputekeeper)
	return
}

func (s *E2ETestSuite) registryKeeper() (queryClient registrytypes.QueryClient, msgServer registrytypes.MsgServer) {
	registrytypes.RegisterQueryServer(s.queryHelper, s.registrykeeper)
	registrytypes.RegisterInterfaces(s.interfaceRegistry)
	queryClient = registrytypes.NewQueryClient(s.queryHelper)
	msgServer = registrykeeper.NewMsgServerImpl(s.registrykeeper)
	return
}

func (s *E2ETestSuite) TestRegisterCommitSubmit() {
	require := s.Require()

	// set up keepers and msg servers
	oraclekeeper, msgServerOracle := s.oracleKeeper()
	require.NotNil(s.T(), msgServerOracle)
	require.NotNil(s.T(), oraclekeeper)
	disputekeeper, msgServerDispute := s.disputeKeeper()
	require.NotNil(s.T(), msgServerDispute)
	require.NotNil(s.T(), disputekeeper)
	registrykeeper, msgServerRegistry := s.registryKeeper()
	require.NotNil(s.T(), msgServerRegistry)
	require.NotNil(s.T(), registrykeeper)

	// register a spec spec1
	spec1 := registrytypes.DataSpec{DocumentHash: "hash1", ValueType: "uint256", AggregationMethod: "weighted-median"}
	specInput := &registrytypes.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "NewQueryType",
		Spec:      spec1,
	}
	registerSpecResult, err := msgServerRegistry.RegisterSpec(s.ctx, specInput)
	require.NoError(err)
	require.NotNil(s.T(), registerSpecResult)

	// register query for spec1
	queryInput := &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "NewQueryType",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err := msgServerRegistry.RegisterQuery(s.ctx, queryInput)
	require.NoError(err)
	require.NotNil(s.T(), registerQueryResult)
	unwrappedCtx := sdk.UnwrapSDKContext(s.ctx)
	queryData, err := registrykeeper.GetQueryData(unwrappedCtx, &types.QueryGetQueryDataRequest{QueryId: registerQueryResult.QueryId})
	require.NoError(err)
	require.NotNil(s.T(), queryData)

	// create account that will become validator
	accAddr, valPrivKey, valPubKey := s.newKeysWithTokens()
	account := authtypes.BaseAccount{
		Address: accAddr.String(),
		PubKey:  codectypes.UnsafePackAny(valPubKey),
	}
	s.accountKeeper.SetAccount(s.ctx, &account)
	valAddr := sdk.ValAddress(accAddr)

	// stake the validator
	val, err := stakingtypes.NewValidator(valAddr, valPubKey, stakingtypes.Description{})
	require.NoError(err)
	s.stakingKeeper.SetValidator(s.ctx, val)
	s.stakingKeeper.SetValidatorByConsAddr(s.ctx, val)
	s.stakingKeeper.SetValidatorByPowerIndex(s.ctx, val)
	_, err = s.stakingKeeper.Delegate(s.ctx, accAddr, sdk.NewInt(1000000), stakingtypes.Unbonded, val, true)
	require.NoError(err)
	_ = staking.EndBlocker(s.ctx, s.stakingKeeper) // updates validator set
	status := s.stakingKeeper.Validator(s.ctx, valAddr).GetStatus()
	require.Equal(stakingtypes.Bonded.String(), status.String())

	// create commit contents
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq oracletypes.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value)
	require.Nil(err)
	signature, err := valPrivKey.Sign(valueDecoded)
	require.Nil(err)
	require.NotNil(s.T(), signature)

	// set commit contents
	commitreq.Creator = accAddr.String()
	commitreq.QueryData = queryData.QueryData
	commitreq.Signature = hex.EncodeToString(signature)

	// commit report
	_, err = msgServerOracle.CommitReport(sdk.WrapSDKContext(s.ctx), &commitreq)
	require.Nil(err)
	_hexxy, _ := hex.DecodeString(queryData.QueryData)

	// get commit value
	commitValue, err := s.oraclekeeper.GetSignature(s.ctx, sdk.AccAddress(valAddr), keeper.HashQueryData(_hexxy))
	fmt.Println("commitValue: ", commitValue)
	require.Nil(err)
	require.NotNil(s.T(), commitValue)

	// verify commit
	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	require.Equal(true, s.oraclekeeper.VerifySignature(s.ctx, accAddr.String(), value, commitValue.Report.Signature))
	require.Equal(commitValue.Report.Creator, accAddr.String())

	reportFromQiD, err := s.oraclekeeper.GetReportsbyQid(ctx, &oracletypes.QueryGetReportsbyQidRequest{QueryId: registerQueryResult.QueryId})
	require.Nil(err)
	fmt.Println("reportFromQiD: ", reportFromQiD) // empty right now ?

	var submitreq oracletypes.MsgSubmitValue
	var submitres oracletypes.MsgSubmitValueResponse
	// forward block by 1 and reveal value
	height := s.ctx.BlockHeight() + 1
	s.ctx = s.ctx.WithBlockHeight(height)
	// Submit value transaction with value revealed, this checks if the value is correctly signed
	submitreq.Creator = accAddr.String()
	submitreq.QueryData = queryData.QueryData
	submitreq.Value = value
	res, err := msgServerOracle.SubmitValue(sdk.WrapSDKContext(s.ctx), &submitreq)
	require.Equal(&submitres, res)
	require.Nil(err)
	report, err := oraclekeeper.GetReportsbyQid(s.ctx, &oracletypes.QueryGetReportsbyQidRequest{QueryId: registerQueryResult.QueryId})
	require.Nil(err)
	fmt.Println("report: ", report)
	expectedPower := sdk.TokensToConsensusPower(sdk.NewInt(1000000), sdk.DefaultPowerReduction)

	microReport := oracletypes.MicroReport{
		Reporter:        accAddr.String(),
		Power:           expectedPower,
		QueryType:       "NewQueryType",
		QueryId:         registerQueryResult.QueryId,
		AggregateMethod: "weighted-median",
		Value:           value,
		BlockNumber:     s.ctx.BlockHeight(),
		Timestamp:       s.ctx.BlockTime(),
	}
	expectedReport := oracletypes.QueryGetReportsbyQidResponse{
		Reports: oracletypes.Reports{
			MicroReports: []*oracletypes.MicroReport{&microReport},
		},
	}
	require.Equal(&expectedReport, report)

	// create dispute
	var disputeReq disputetypes.MsgDispute

}
