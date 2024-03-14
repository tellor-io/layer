package e2e_test

import (
	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
)

func (s *E2ETestSuite) TestInitialMint() {
	require := s.Require()

	mintToTeamAcc := s.accountKeeper.GetModuleAddress(minttypes.MintToTeam)
	require.NotNil(mintToTeamAcc)
	balance := s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom)
	require.Equal(balance.Amount, math.NewInt(300*1e6))
}

func (s *E2ETestSuite) TestTransfer() {
	require := s.Require()

	mintToTeamAcc := s.accountKeeper.GetModuleAddress(minttypes.MintToTeam)
	require.NotNil(mintToTeamAcc)
	balance := s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom)
	require.Equal(balance.Amount, math.NewInt(300*1e6))

	// create 10 accounts
	type Accounts struct {
		PrivateKey secp256k1.PrivKey
		Account    sdk.AccAddress
	}
	accounts := make([]Accounts, 0, 10)

	// transfer from team to 10 accounts
	for i := 0; i < 10; i++ {
		privKey := secp256k1.GenPrivKey()
		accountAddress := sdk.AccAddress(privKey.PubKey().Address())
		account := authtypes.BaseAccount{
			Address:       accountAddress.String(),
			PubKey:        codectypes.UnsafePackAny(privKey.PubKey()),
			AccountNumber: uint64(i + 1),
		}
		existingAccount := s.accountKeeper.GetAccount(s.ctx, accountAddress)
		if existingAccount == nil {
			s.accountKeeper.SetAccount(s.ctx, &account)
			accounts = append(accounts, Accounts{
				PrivateKey: *privKey,
				Account:    accountAddress,
			})
		}
	}

	for _, acc := range accounts {
		startBalance := s.bankKeeper.GetBalance(s.ctx, acc.Account, s.denom).Amount
		err := s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, minttypes.MintToTeam, acc.Account, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
		require.NoError(err)
		require.Equal(startBalance.Add(math.NewInt(1000)), s.bankKeeper.GetBalance(s.ctx, acc.Account, s.denom).Amount)
	}
	expectedTeamBalance := math.NewInt(300*1e6 - 1000*10)
	require.Equal(expectedTeamBalance, s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom).Amount)

	//transfer from account 0 to account 1
	s.bankKeeper.SendCoins(s.ctx, accounts[0].Account, accounts[1].Account, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
	require.Equal(math.NewInt(0), s.bankKeeper.GetBalance(s.ctx, accounts[0].Account, s.denom).Amount)
	require.Equal(math.NewInt(2000), s.bankKeeper.GetBalance(s.ctx, accounts[1].Account, s.denom).Amount)

	//transfer from account 2 to team
	s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, accounts[2].Account, minttypes.MintToTeam, sdk.NewCoins(sdk.NewCoin(s.denom, math.NewInt(1000))))
	require.Equal(math.NewInt(0), s.bankKeeper.GetBalance(s.ctx, accounts[2].Account, s.denom).Amount)
	require.Equal(expectedTeamBalance.Add(math.NewInt(1000)), s.bankKeeper.GetBalance(s.ctx, mintToTeamAcc, s.denom).Amount)
}

// func (s *E2ETestSuite) TestStakeTokens() {
// 	require := s.Require()

// 	accountAddrs, validatorAddrs := s.createValidators([]int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100})
// 	for i := range accountAddrs {
// 		validator, err := s.stakingKeeper.Validator(s.ctx, validatorAddrs[i])
// 		status := validator.GetStatus()
// 		require.Nil(err)
// 		require.Equal(stakingtypes.Bonded.String(), status.String())
// 	}

// 	// self-delegate
// 	val, err := s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	power := val.GetConsensusPower(sdk.DefaultPowerReduction) // start with 10
// 	require.Equal(math.NewInt(10), math.NewInt(power))
// 	_, err = s.stakingKeeper.Delegate(s.ctx, accountAddrs[0], math.NewInt(10*1e6), stakingtypes.Unbonded, val, true) // delegate 10
// 	require.Nil(err)
// 	val, err = s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	actualPower := val.GetConsensusPower(sdk.DefaultPowerReduction)                                                                  // 20
// 	expectedPower := math.NewInt(power).Add(math.NewInt(sdk.TokensToConsensusPower(math.NewInt(10*1e6), sdk.DefaultPowerReduction))) // 20
// 	require.Equal(expectedPower, math.NewInt(actualPower))

// 	// undelegate 1 of self-delegated stake
// 	power = val.GetConsensusPower(sdk.DefaultPowerReduction) // 20
// 	sharesAmount, err := s.stakingKeeper.ValidateUnbondAmount(
// 		s.ctx, accountAddrs[1], validatorAddrs[0], math.NewInt(10*1e5),
// 	)
// 	require.Nil(err)
// 	_, _, err = s.stakingKeeper.Undelegate(s.ctx, accountAddrs[0], validatorAddrs[0], sharesAmount) // undelegate 1
// 	require.Nil(err)

// 	unbondingAmount, err := s.stakingKeeper.GetDelegatorUnbonding(s.ctx, accountAddrs[0])
// 	fmt.Println("unbondingAmount: ", unbondingAmount)
// 	require.Nil(err)
// 	currentTime := s.ctx.BlockTime()
// 	fmt.Println("current time: ", currentTime)
// 	unbondingDelegation, err := s.stakingKeeper.GetAllUnbondingDelegations(s.ctx, accountAddrs[0])
// 	require.Nil(err)
// 	fmt.Println("unbondingDelegation: ", unbondingDelegation)

// 	val, err = s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	actualPower = val.GetConsensusPower(sdk.DefaultPowerReduction)                                                                  // 19
// 	expectedPower = math.NewInt(power).Sub(math.NewInt(sdk.TokensToConsensusPower(math.NewInt(10*1e5), sdk.DefaultPowerReduction))) // 19
// 	require.Equal(expectedPower, math.NewInt(actualPower))

// 	// delegate from validator 1 to validator 0
// 	val, err = s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	power = val.GetConsensusPower(sdk.DefaultPowerReduction)                                                         // 19
// 	_, err = s.stakingKeeper.Delegate(s.ctx, accountAddrs[1], math.NewInt(10*1e6), stakingtypes.Unbonded, val, true) // delegate 10
// 	require.Nil(err)
// 	val, err = s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	actualPower = val.GetConsensusPower(sdk.DefaultPowerReduction)                                                                  // 29
// 	expectedPower = math.NewInt(power).Add(math.NewInt(sdk.TokensToConsensusPower(math.NewInt(10*1e6), sdk.DefaultPowerReduction))) // 29
// 	require.Equal(expectedPower, math.NewInt(actualPower))

// 	// undelegate from validator 1 to validator 0
// 	power = val.GetConsensusPower(sdk.DefaultPowerReduction) // 29
// 	sharesAmount, err = s.stakingKeeper.ValidateUnbondAmount(
// 		s.ctx, accountAddrs[1], validatorAddrs[0], math.NewInt(10*1e5),
// 	)
// 	require.Nil(err)
// 	// sharesAmount = math.LegacyNewDecFromInt(math.NewInt(1))
// 	_, _, err = s.stakingKeeper.Undelegate(s.ctx, accountAddrs[1], validatorAddrs[0], sharesAmount)
// 	require.Nil(err)

// 	unbondingAmount, err = s.stakingKeeper.GetDelegatorUnbonding(s.ctx, accountAddrs[1])
// 	fmt.Println("unbondingAmount: ", unbondingAmount)
// 	require.Nil(err)
// 	currentTime = s.ctx.BlockTime()
// 	fmt.Println("current time: ", currentTime)
// 	unbondingDelegation, err = s.stakingKeeper.GetAllUnbondingDelegations(s.ctx, accountAddrs[1])
// 	require.Nil(err)
// 	fmt.Println("unbondingDelegation: ", unbondingDelegation)

// 	val, err = s.stakingKeeper.GetValidator(s.ctx, validatorAddrs[0])
// 	require.Nil(err)
// 	actualPower = val.GetConsensusPower(sdk.DefaultPowerReduction) // should be 28 ?
// 	fmt.Println("actual power: ", actualPower)
// 	expectedPower = math.NewInt(power).Sub(math.NewInt(sdk.TokensToConsensusPower(math.NewInt(10*1e5), sdk.DefaultPowerReduction))) // 28
// 	fmt.Println("expected power: ", expectedPower)
// 	require.Equal(expectedPower, math.NewInt(actualPower))

// }

func (s *E2ETestSuite) TestReporterJail() {
	// require := s.Require()

	// create validators
	// create reporters

	// report for whatever is in cycle list

	// currentTime := s.ctx.BlockTime()
	// fmt.Println(currentTime)
	// s.ctx = s.ctx.WithBlockTime(currentTime.Add(600 * time.Second)) // add 10 minutes
	// newTime := s.ctx.BlockTime()
	// fmt.Println(newTime)

}

func (s *E2ETestSuite) TestValidateCycleList() {
	require := s.Require()

	// block 0
	_, err := s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	firstQuery := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.Equal(btcQueryData[2:], firstQuery)
	require.Equal(s.ctx.BlockHeight(), int64(0))

	// block 1
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(1))
	secondQuery := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.Equal(trbQueryData[2:], secondQuery)

	// block 2
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	_, err = s.app.BeginBlocker(s.ctx)
	require.NoError(err)
	require.Equal(s.ctx.BlockHeight(), int64(2))
	thirdQuery := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	require.Equal(ethQueryData[2:], thirdQuery)
}

func (s *E2ETestSuite) TestSubmit() {
	// require := s.Require()
	// _, msgServerOracle := s.oracleKeeper()
	// require.NotNil(msgServerOracle)
	// currentQuery := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	// queryDataBytes, err := hex.DecodeString(currentQuery[2:])
	// require.Nil(err)
	// _ = crypto.Keccak256(queryDataBytes)
	// // queryId := hex.EncodeToString(queryIdBytes)

	// accountAddrs, validatorAddrs := s.createValidators([]int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100})
	// for i := range accountAddrs {
	// 	validator, err := s.stakingKeeper.Validator(s.ctx, validatorAddrs[i])
	// 	status := validator.GetStatus()
	// 	require.Nil(err)
	// 	require.Equal(stakingtypes.Bonded.String(), status.String())
	// }

	// // commit
	// err = CommitReport(s.ctx, string(accountAddrs[0].String()), currentQuery, msgServerOracle)
	// require.Nil(err)

	// commit, err := s.oraclekeeper.GetCommit(s.ctx, accountAddrs[0], queryIdBytes)
	// require.Nil(err)
	// require.NotNil(commit)
	// fmt.Println("commit: ", commit)

	// value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	// valueDecoded, err := hex.DecodeString(value) // convert hex value to bytes
	// s.Nil(err)
	// salt, err := utils.Salt(32)
	// s.Nil(err)
	// hash := utils.CalculateCommitment(string(valueDecoded), salt)
	// s.Nil(err)
	// // commit report with query data in cycle list
	// commitreq := &oracletypes.MsgCommitReport{
	// 	Creator:   accountAddrs[0].String(),
	// 	QueryData: currentQuery,
	// 	Hash:      hash,
	// }
	// _, err = msgServerOracle.CommitReport(s.ctx, commitreq)
	// require.Nil(err)

	// // submit
	// var submitreq types.MsgSubmitValue
	// var submitres types.MsgSubmitValueResponse

	// height := s.ctx.BlockHeight() + 1
	// s.ctx = s.ctx.WithBlockHeight(height)
	// // Submit value transaction with value revealed, this checks if the value is correctly hashed
	// submitreq.Creator = Addr.String()
	// submitreq.QueryData = queryData
	// submitreq.Value = value
	// submitreq.Salt = salt
	// res, err := s.msgServer.SubmitValue(s.ctx, &submitreq)
	// require.Equal(&submitres, res)
	// require.Nil(err)
	// report, err := s.oracleKeeper.GetReportsbyQid(s.ctx, &types.QueryGetReportsbyQidRequest{QueryId: "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"})
	// require.Nil(err)
	// microReport := types.MicroReport{
	// 	Reporter:        Addr.String(),
	// 	Power:           1000000000000,
	// 	QueryType:       "SpotPrice",
	// 	QueryId:         "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
	// 	AggregateMethod: "weighted-median",
	// 	Value:           value,
	// 	BlockNumber:     s.ctx.BlockHeight(),
	// 	Timestamp:       s.ctx.BlockTime(),
	// }
	// expectedReport := types.QueryGetReportsbyQidResponse{
	// 	Reports: types.Reports{
	// 		MicroReports: []*types.MicroReport{&microReport},
	// 	},
	// }
	// require.Equal(&expectedReport, report)

}

func (s *E2ETestSuite) TestDisputes() {
	require := s.Require()
	_, msgServerDispute := s.disputeKeeper()
	require.NotNil(msgServerDispute)

	// // create dispute
	// var disputeReq disputetypes.MsgDispute
	// var disputeRes disputetypes.MsgDisputeResponse
	// disputeReq.Creator = accountAddrs[0].String()
	// disputeReq.QueryId = currentQuery.QueryId
	// disputeReq.DisputeType = "query"
	// disputeReq.DisputeId = "1"
	// disputeReq.Value = ""
}

// get delegation
// call slash

func (s *E2ETestSuite) TestTipCommitReveal() {
	// require := s.Require()

	// // set up keepers and msg servers
	// oraclekeeper, msgServerOracle := s.oracleKeeper()
	// require.NotNil(msgServerOracle)
	// require.NotNil(oraclekeeper)
	// disputekeeper, msgServerDispute := s.disputeKeeper()
	// require.NotNil(msgServerDispute)
	// require.NotNil(disputekeeper)
	// registrykeeper, msgServerRegistry := s.registryKeeper()
	// require.NotNil(msgServerRegistry)
	// require.NotNil(registrykeeper)

	// // register a spec spec1
	// spec1 := registrytypes.DataSpec{DocumentHash: "hash1", ResponseValueType: "uint256", AggregationMethod: "weighted-median"}
	// specInput := &registrytypes.MsgRegisterSpec{
	// 	Registrar: "creator1",
	// 	QueryType: "NewQueryType",
	// 	Spec:      spec1,
	// }
	// registerSpecResult, err := msgServerRegistry.RegisterSpec(s.ctx, specInput)
	// require.NoError(err)
	// require.NotNil(s.T(), registerSpecResult)

	// // create account that will become validator
	// accAddr, valPrivKey, valPubKey := s.newKeysWithTokens()
	// account := authtypes.BaseAccount{
	// 	Address: accAddr.String(),
	// 	PubKey:  codectypes.UnsafePackAny(valPubKey),
	// }
	// s.accountKeeper.SetAccount(s.ctx, &account)
	// valAddr := sdk.ValAddress(accAddr)

	// // stake the validator
	// val, err := stakingtypes.NewValidator(valAddr.String(), valPubKey, stakingtypes.Description{})
	// require.NoError(err)
	// s.stakingKeeper.SetValidator(s.ctx, val)
	// s.stakingKeeper.SetValidatorByConsAddr(s.ctx, val)
	// s.stakingKeeper.SetValidatorByPowerIndex(s.ctx, val)
	// _, err = s.stakingKeeper.Delegate(s.ctx, accAddr, math.NewInt(1000000), stakingtypes.Unbonded, val, true)
	// require.NoError(err)
	// _ = sdk.EndBlocker(s.app.EndBlocker) // updates validator set
	// validator, err := s.stakingKeeper.Validator(s.ctx, valAddr)
	// require.NoError(err)
	// status := validator.GetStatus()
	// require.Equal(stakingtypes.Bonded.String(), status.String())

	// // create commit contents
	// value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	// // var commitreq oracletypes.MsgCommitReport
	// valueDecoded, err := hex.DecodeString(value)
	// require.Nil(err)
	// signature, err := valPrivKey.Sign(valueDecoded)
	// require.Nil(err)
	// require.NotNil(s.T(), signature)

	// set commit contents
	// commitreq.Creator = accAddr.String()
	// commitreq.QueryData = queryData.QueryData
	// commitreq.Hash = hex.EncodeToString(signature)

	// // commit report
	// _, err = msgServerOracle.CommitReport(s.ctx, &commitreq)
	// require.Nil(err)
	// _hexxy, _ := hex.DecodeString(queryData.QueryData)

	// // get commit value
	// commitValue, err := s.oraclekeeper.GetCommit(s.ctx, sdk.AccAddress(valAddr), keeper.HashQueryData(_hexxy))
	// fmt.Println("commitValue: ", commitValue)
	// require.Nil(err)
	// require.NotNil(s.T(), commitValue)

	// // verify commit
	// ctx := s.ctx.WithBlockTime(s.ctx.BlockTime().Add(86400*2 + 1))
	// require.Equal(true, s.oraclekeeper.VerifySignature(s.ctx, accAddr.String(), value, commitValue.Report.Hash))
	// require.Equal(commitValue.Report.Creator, accAddr.String())

	// reportFromQiD, err := s.oraclekeeper.GetReportsbyQid(ctx, &oracletypes.QueryGetReportsbyQidRequest{QueryId: registerQueryResult.QueryId})
	// require.Nil(err)
	// fmt.Println("reportFromQiD: ", reportFromQiD) // empty right now ?

	// var submitreq oracletypes.MsgSubmitValue
	// var submitres oracletypes.MsgSubmitValueResponse
	// // forward block by 1 and reveal value
	// height := s.ctx.BlockHeight() + 1
	// s.ctx = s.ctx.WithBlockHeight(height)
	// // Submit value transaction with value revealed, this checks if the value is correctly signed
	// submitreq.Creator = accAddr.String()
	// submitreq.QueryData = queryData.QueryData
	// submitreq.Value = value
	// res, err := msgServerOracle.SubmitValue(sdk.WrapSDKContext(s.ctx), &submitreq)
	// require.Equal(&submitres, res)
	// require.Nil(err)
	// report, err := oraclekeeper.GetReportsbyQid(s.ctx, &oracletypes.QueryGetReportsbyQidRequest{QueryId: registerQueryResult.QueryId})
	// require.Nil(err)
	// fmt.Println("report: ", report)
	// expectedPower := sdk.TokensToConsensusPower(math.NewInt(1000000), sdk.DefaultPowerReduction)

	// microReport := oracletypes.MicroReport{
	// 	Reporter:        accAddr.String(),
	// 	Power:           expectedPower,
	// 	QueryType:       "NewQueryType",
	// 	QueryId:         registerQueryResult.QueryId,
	// 	AggregateMethod: "weighted-median",
	// 	Value:           value,
	// 	BlockNumber:     s.ctx.BlockHeight(),
	// 	Timestamp:       s.ctx.BlockTime(),
	// }
	// expectedReport := oracletypes.QueryGetReportsbyQidResponse{
	// 	Reports: oracletypes.Reports{
	// 		MicroReports: []*oracletypes.MicroReport{&microReport},
	// 	},
	// }
	// require.Equal(&expectedReport, report)

	// create dispute
	// var disputeReq disputetypes.MsgDispute
}
