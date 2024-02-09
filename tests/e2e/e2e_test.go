package e2e_test

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/registry/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
)

// func (s *E2ETestSuite) TestMint500kToVal() []sdk.AccAddress {
// 	// require := s.Require()

// 	// // Create 10 validators
// 	// powers := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
// 	// valSet := s.createValidatorAccs(powers)
// 	// target := valSet[9]
// 	// startingBalance := s.bankKeeper.GetBalance(s.ctx, target, "loya").Amount

// 	// // Mint coins to module account
// 	// mintAmount := math.NewInt(500000)
// 	// coin := sdk.NewCoin(s.denom, mintAmount)
// 	// err := s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(coin))
// 	// require.NoError(err)

// 	// // transfer coins from module account to target
// 	// err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, target, sdk.NewCoins(coin))
// 	// require.NoError(err)

// 	// // Check that the recipient's balance was updated
// 	// balance := s.bankKeeper.GetBalance(s.ctx, target, "loya").Amount
// 	// require.Equal(startingBalance.Add(mintAmount), balance)
// 	// return valSet
// }

func (s *E2ETestSuite) TestMint500kToAccount() {
	// require := s.Require()

	// // Create an account
	// PrivKey := secp256k1.GenPrivKey()
	// PubKey := PrivKey.PubKey()
	// Addr := sdk.AccAddress(PubKey.Address())

	// // set account with auth module
	// acc := authtypes.BaseAccount{
	// 	Address: Addr.String(),
	// 	PubKey:  codectypes.UnsafePackAny(PubKey),
	// }
	// s.accountKeeper.SetAccount(s.ctx, &acc)
	// startingBalance := s.bankKeeper.GetBalance(s.ctx, Addr, "loya").Amount
	// require.Equal(math.NewInt(0), startingBalance)

	// // mint to module
	// mintAmount := math.NewInt(500000)
	// coin := sdk.NewCoin(s.denom, mintAmount)
	// err := s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(coin))
	// require.NoError(err)

	// // transfer coins from module to account
	// err = s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, Addr, sdk.NewCoins(coin))
	// require.NoError(err)

	// // Check that the balance was updated
	// balance := s.bankKeeper.GetBalance(s.ctx, Addr, "loya").Amount
	// require.Equal(startingBalance.Add(mintAmount), balance)
}

// func (s *E2ETestSuite) TestTransferFromValToVal() {
// 	require := s.Require()

// 	valSet := s.TestMint500kToVal()
// 	val := valSet[9]
// 	friends := valSet[:9]
// 	transferAmount := math.NewInt(100)
// 	coin := sdk.NewCoin(s.denom, transferAmount)

// 	// Transfer coins to each friend
// 	for _, friend := range friends {
// 		startBalance := s.bankKeeper.GetBalance(s.ctx, friend, "loya").Amount
// 		err := s.bankKeeper.SendCoins(s.ctx, val, friend, sdk.NewCoins(coin))
// 		require.NoError(err)
// 		require.Equal(startBalance.Add(transferAmount), s.bankKeeper.GetBalance(s.ctx, friend, "loya").Amount)
// 	}
// }

// func (s *E2ETestSuite) TestStakeFromVal() {
// 	require := s.Require()

// 	valSet := s.TestMint500kToVal()
// 	val := valSet[9]
// 	validator, err := s.stakingKeeper.Validator(s.ctx, sdk.ValAddress(val))
// 	require.NoError(err)
// 	startingPower := validator.GetBondedTokens()
// 	fmt.Println("startingPower: ", startingPower)

// }

// func (s *E2ETestSuite) TestTransferFromValToAcc() {
// 	require := s.Require()

// 	valSet := s.TestMint500k()
// 	val := valSet[9]
// 	acc, _, _ := s.newKeysWithTokens()
// 	startBalance := s.bankKeeper.GetBalance(s.ctx, acc, "loya").Amount
// 	transferAmount := math.NewInt(100)
// 	coin := sdk.NewCoin(s.denom, transferAmount)
// 	err := s.bankKeeper.SendCoins(s.ctx, val, acc, sdk.NewCoins(coin))
// 	require.NoError(err)
// 	require.Equal(startBalance.Add(transferAmount), s.bankKeeper.GetBalance(s.ctx, acc, "loya").Amount)
// }

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
	val, err := stakingtypes.NewValidator(valAddr.String(), valPubKey, stakingtypes.Description{})
	require.NoError(err)
	s.stakingKeeper.SetValidator(s.ctx, val)
	s.stakingKeeper.SetValidatorByConsAddr(s.ctx, val)
	s.stakingKeeper.SetValidatorByPowerIndex(s.ctx, val)
	_, err = s.stakingKeeper.Delegate(s.ctx, accAddr, math.NewInt(1000000), stakingtypes.Unbonded, val, true)
	require.NoError(err)
	_ = sdk.EndBlocker(s.app.EndBlocker) // updates validator set
	validator, err := s.stakingKeeper.Validator(s.ctx, valAddr)
	status := validator.GetStatus()
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
	commitreq.Hash = hex.EncodeToString(signature)

	// commit report
	_, err = msgServerOracle.CommitReport(s.ctx, &commitreq)
	require.Nil(err)
	_hexxy, _ := hex.DecodeString(queryData.QueryData)

	// get commit value
	commitValue, err := s.oraclekeeper.GetCommit(s.ctx, sdk.AccAddress(valAddr), keeper.HashQueryData(_hexxy))
	fmt.Println("commitValue: ", commitValue)
	require.Nil(err)
	require.NotNil(s.T(), commitValue)

	// verify commit
	ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	require.Equal(true, s.oraclekeeper.VerifySignature(s.ctx, accAddr.String(), value, commitValue.Report.Hash))
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
	expectedPower := sdk.TokensToConsensusPower(math.NewInt(1000000), sdk.DefaultPowerReduction)

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
	// var disputeReq disputetypes.MsgDispute
}
