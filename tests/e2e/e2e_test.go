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

func (s *E2ETestSuite) TestRegisterSubmitDispute() {
	require := s.Require()

	// set up keepers and msg servers
	kOracle, msgServerOracle := s.oracleKeeper()
	require.NotNil(s.T(), msgServerOracle)
	require.NotNil(s.T(), kOracle)
	kDispute, msgServerDispute := s.disputeKeeper()
	require.NotNil(s.T(), msgServerDispute)
	require.NotNil(s.T(), kDispute)
	kRegistry, msgServerRegistry := s.registryKeeper()
	require.NotNil(s.T(), msgServerRegistry)
	require.NotNil(s.T(), kRegistry)

	// register a spec spec1
	spec1 := registrytypes.DataSpec{DocumentHash: "hash1", ValueType: "uint256"}
	specInput := &registrytypes.MsgRegisterSpec{
		Creator:   "creator1",
		QueryType: "queryType1",
		Spec:      spec1,
	}
	registerSpecResult, err := msgServerRegistry.RegisterSpec(s.ctx, specInput)
	require.NoError(err)
	require.NotNil(s.T(), registerSpecResult)

	// register query for spec1
	queryInput := &types.MsgRegisterQuery{
		Creator:    "creator1",
		QueryType:  "queryType1",
		DataTypes:  []string{"uint256", "uint256"},
		DataFields: []string{"1", "2"},
	}
	registerQueryResult, err := msgServerRegistry.RegisterQuery(s.ctx, queryInput)
	require.NoError(err)
	require.NotNil(s.T(), registerQueryResult)
	unwrappedCtx := sdk.UnwrapSDKContext(s.ctx)
	queryData, err := kRegistry.GetQueryData(unwrappedCtx, &types.QueryGetQueryDataRequest{QueryId: registerQueryResult.QueryId})
	require.NoError(err)
	require.NotNil(s.T(), queryData)

	// use create validators

	// use one of the accounts it gives

	// get account with tokens
	accAddr, valPrivKey, valPubKey := s.newKeysWithTokens()
	account := authtypes.BaseAccount{
		Address: accAddr.String(),
		PubKey:  codectypes.UnsafePackAny(valPubKey),
	}
	s.accountKeeper.SetAccount(s.ctx, &account)
	valAddr := sdk.ValAddress(accAddr)
	// stake tokens
	val, err := stakingtypes.NewValidator(valAddr, valPubKey, stakingtypes.Description{})
	require.NoError(err)
	s.stakingKeeper.SetValidator(s.ctx, val)
	s.stakingKeeper.SetValidatorByConsAddr(s.ctx, val)
	s.stakingKeeper.SetValidatorByPowerIndex(s.ctx, val)
	_, err = s.stakingKeeper.Delegate(s.ctx, accAddr, sdk.NewInt(1000000), stakingtypes.Unbonded, val, true)
	require.NoError(err)
	_ = staking.EndBlocker(s.ctx, s.stakingKeeper) // updates

	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	status := s.stakingKeeper.Validator(ctx, valAddr).GetStatus()
	fmt.Println("val after: ", status)

	// create commit contents
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	var commitreq oracletypes.MsgCommitReport
	valueDecoded, err := hex.DecodeString(value) // convert hex value to bytes
	require.Nil(err)
	signature, err := valPrivKey.Sign(valueDecoded) // sign value
	require.Nil(err)
	require.NotNil(s.T(), signature)

	// set commit contents
	commitreq.Creator = accAddr.String()                // set creator to val2 address
	commitreq.QueryData = queryData.QueryData           // set query data to query data from query1
	commitreq.Signature = hex.EncodeToString(signature) // set commit signature
	// commit data
	_, err = msgServerOracle.CommitReport(sdk.WrapSDKContext(s.ctx), &commitreq)
	require.Nil(err)
	_hexxy, _ := hex.DecodeString(queryData.QueryData)
	// get signature from commit
	commitValue, err := s.oraclekeeper.GetSignature(s.ctx, sdk.AccAddress(valAddr), keeper.HashQueryData(_hexxy))
	fmt.Println("commitValue: ", commitValue)
	require.Nil(err)
	require.NotNil(s.T(), commitValue)
	// verify report signature
	require.Equal(true, s.oraclekeeper.VerifySignature(s.ctx, sdk.AccAddress(valAddr).String(), value, commitValue.Report.Signature))

	fmt.Println("sdk.AccAddress(val2).String(): ", sdk.AccAddress(valAddr).String())
	fmt.Println("commitValue.Report.Signature: ", commitValue.Report.Signature)
	require.Equal(commitValue.Report.Creator, sdk.AccAddress(valAddr).String())

	// //forward block
	// ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	// fmt.Println("ctx: ", ctx)

	// // dispute that value

}
