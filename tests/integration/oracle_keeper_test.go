package integration_test

import (
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"

	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

func (s *IntegrationTestSuite) TestTipping() {
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.oraclekeeper.GetQueryTip(s.ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	userTips, err := s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.NoError(err)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total.Int64(), tips.Int64())

	// tip same query again
	_, err = msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	tips, err = s.oraclekeeper.GetQueryTip(s.ctx, queryId)
	s.NoError(err)
	// tips should be 2x
	s.Equal(tip.Sub(twoPercent).Amount.Mul(math.NewInt(2)), tips)

	// total tips overall
	userTips, err = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.NoError(err)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips)

	// // tip different query
	// btcQueryId := utils.QueryIDFromData(btcQueryData)

	// _, err = msgServer.Tip(s.ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	// s.NoError(err)
	// tips, err = s.oraclekeeper.GetQueryTip(s.ctx, btcQueryId)
	// s.NoError(err)
	// s.Equal(tip.Sub(twoPercent).Amount, tips)

	// userQueryTips, _ := s.oraclekeeper.Tips.Get(s.ctx, collections.Join(btcQueryId, addr.Bytes()))
	// s.Equal(userQueryTips, tips)
	// userTips, err = s.oraclekeeper.GetUserTips(s.ctx, addr)
	// s.NoError(err)
	// s.Equal(userTips.Address, addr.String())
	// s.Equal(userTips.Total, tips.Add(tips).Add(tips))
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetCurrentTip(s.ctx, &types.QueryGetCurrentTipRequest{QueryData: ethQueryData})
	s.NoError(err)
	s.Equal(resp.Tips, tip.Amount.Sub(twoPercent.Amount))
}

// test tipping, reporting and allocation of rewards
func (s *IntegrationTestSuite) TestTippingReporting() {
	s.ctx = s.ctx.WithBlockTime(time.Now())
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.oraclekeeper.GetQueryTip(s.ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	newReporter, err := createReporter(s.ctx, int64(1), s.reporterkeeper)
	s.Nil(err)
	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	value := encodeValue(29266)
	hash := oracleutils.CalculateCommitment(value, salt)
	commit, reveal := report(newReporter.String(), value, salt, hash, ethQueryData)
	_, err = msgServer.CommitReport(s.ctx, &commit)
	s.Nil(err)
	_, err = msgServer.SubmitValue(s.ctx, &reveal)
	s.Nil(err)
	// advance time to expire the query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second * 7)) // bypassing offset that expires time to commit/reveal
	err = s.oraclekeeper.SetAggregatedReport(s.ctx)
	s.Nil(err)
	res, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: queryId})
	s.Nil(err)
	s.Equal(res.Report.AggregateReporter, newReporter.String())
	// tip should be 0 after aggregated report
	tips, err = s.oraclekeeper.GetQueryTip(s.ctx, queryId)
	s.Nil(err)
	s.Equal(tips, math.ZeroInt())
	totalTips, err := s.oraclekeeper.GetTotalTips(s.ctx)
	s.Nil(err)
	s.Equal(totalTips, tip.Sub(twoPercent).Amount) // total tips should be equal to the tipped amount minus 2% burned
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := math.NewInt(1000)
	twoPercent := tip.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.denom, tip),
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String(), QueryData: ethQueryData})
	s.NoError(err)
	s.Equal(resp.TotalTips.Total, tip.Sub(twoPercent))
	// Check total tips without a given query data
	resp, err = s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestSmallTip() {
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.denom, math.NewInt(10))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	accBalanceBefore := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)
	modBalanceBefore := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), s.denom)
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	accBalanceAfter := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)
	modBalanceAfter := s.bankKeeper.GetBalance(s.ctx, authtypes.NewModuleAddress(types.ModuleName), s.denom)
	s.Equal(accBalanceBefore.Amount.Sub(tip.Amount), accBalanceAfter.Amount)
	s.Equal(modBalanceBefore.Amount.Add(tip.Amount).Sub(twoPercent.Amount), modBalanceAfter.Amount)
}

func (s *IntegrationTestSuite) TestMedianReports() {
	s.ctx = s.ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)
	tipper := s.newKeysWithTokens()

	reporters := []struct {
		name          string
		reporterIndex int
		value         string
		stakeAmount   math.Int
		power         int64
	}{
		{
			name:          "reporter 1",
			reporterIndex: 0,
			value:         encodeValue(162926),
			stakeAmount:   math.NewInt(1_000_000),
			power:         1,
		},
		{
			name:          "reporter 2",
			reporterIndex: 1,
			value:         encodeValue(362926),
			stakeAmount:   math.NewInt(2_000_000),
			power:         2,
		},
		{
			name:          "reporter 3",
			reporterIndex: 2,
			value:         encodeValue(262926),
			stakeAmount:   math.NewInt(3_000_000),
			power:         3,
		},
		{
			name:          "reporter 4",
			reporterIndex: 3,
			value:         encodeValue(562926),
			stakeAmount:   math.NewInt(4_000_000),
			power:         4,
		},
		{
			name:          "reporter 5",
			reporterIndex: 4,
			value:         encodeValue(462926),
			stakeAmount:   math.NewInt(5_000_000),
			power:         5,
		},
	}
	msgServer.Tip(s.ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: sdk.NewCoin(s.denom, math.NewInt(1000))})
	addr := make([]sdk.AccAddress, len(reporters))
	for _, r := range reporters {
		s.T().Run(r.name, func(t *testing.T) {
			// create reporter
			newReporter, err := createReporter(s.ctx, r.power, s.reporterkeeper)
			s.Nil(err)
			addr[r.reporterIndex] = newReporter
			salt, err := oracleutils.Salt(32)
			s.Nil(err)
			hash := oracleutils.CalculateCommitment(r.value, salt)
			s.Nil(err)
			commit, reveal := report(newReporter.String(), r.value, salt, hash, ethQueryData)
			_, err = msgServer.CommitReport(s.ctx, &commit)
			s.Nil(err)
			_, err = msgServer.SubmitValue(s.ctx, &reveal)
			s.Nil(err)
		})
	}
	// advance time to expire query and aggregate report
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second * 7)) // bypass time to expire query so it can be aggregated
	s.app.EndBlocker(s.ctx)                                             // EndBlocker aggregates reports
	// check median
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	res, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	s.Nil(err)
	expectedMedianReporterIndex := 4
	expectedMedianReporter := addr[expectedMedianReporterIndex].String()
	s.Equal(expectedMedianReporter, res.Report.AggregateReporter)
	s.Equal(reporters[expectedMedianReporterIndex].value, res.Report.AggregateValue)
}

func report(creator, value, salt, hash string, qdata []byte) (types.MsgCommitReport, types.MsgSubmitValue) {
	commit := types.MsgCommitReport{
		Creator:   creator,
		QueryData: qdata,
		Hash:      hash,
	}
	reveal := types.MsgSubmitValue{
		Creator:   creator,
		QueryData: qdata,
		Value:     value,
		Salt:      salt,
	}
	return commit, reveal
}

func (s *IntegrationTestSuite) TestGetCylceListQueries() {
	accs, _, _ := s.createValidatorAccs([]int64{100, 200, 300, 400, 500})
	// Get supported queries
	resp, err := s.oraclekeeper.GetCyclelist(s.ctx)
	s.NoError(err)
	s.Equal(resp, [][]byte{trbQueryData, ethQueryData, btcQueryData})

	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msgContent := &types.MsgUpdateCyclelist{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Cyclelist: [][]byte{matic},
	}
	proposal1, err := s.govKeeper.SubmitProposal(s.ctx, []sdk.Msg{msgContent}, "", "test", "description", accs[0], false)
	s.NoError(err)

	govParams, err := s.govKeeper.Params.Get(s.ctx)
	s.NoError(err)
	votingStarted, err := s.govKeeper.AddDeposit(s.ctx, proposal1.Id, accs[0], govParams.MinDeposit)
	s.NoError(err)
	s.True(votingStarted)
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusVotingPeriod)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[1], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[2], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal1.Id)
	s.NoError(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour * 24 * 2))
	gov.EndBlocker(s.ctx, s.govKeeper)
	proposal1, err = s.govKeeper.Proposals.Get(s.ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusPassed)
	resp, err = s.oraclekeeper.GetCyclelist(s.ctx)
	s.NoError(err)
	s.Equal([][]byte{matic}, resp)
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsOneReporter() {
	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err := s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.denom, reward)))
	s.NoError(err)

	// testing for a query id and check if the reporter gets the reward, bypassing the commit/reveal process
	qId := utils.QueryIDFromData(ethQueryData)
	value := []string{"000001"}

	reporterPower := int64(1)
	addr, err := createReporter(s.ctx, 1, s.reporterkeeper)
	s.NoError(err)

	reports := testutil.GenerateReports([]sdk.AccAddress{addr}, value, []int64{reporterPower}, qId)

	bal1 := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)

	_, err = s.oraclekeeper.WeightedMedian(s.ctx, reports[:1])
	s.NoError(err)
	res, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	s.NoError(err)
	s.Equal(res.Report.AggregateReportIndex, int64(0), "single report should be at index 0")

	tbr, err := s.oraclekeeper.GetTimeBasedRewards(s.ctx, &types.QueryGetTimeBasedRewardsRequest{})
	s.NoError(err)

	err = s.oraclekeeper.AllocateRewards(s.ctx, res.Report.Reporters, tbr.Reward.Amount, false)
	s.NoError(err)
	// advance height
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	amt, err := s.reporterkeeper.WithdrawDelegationRewards(s.ctx, addr.Bytes(), addr)
	s.NoError(err)
	s.True(amt.AmountOf(s.denom).GT(math.ZeroInt()))

	bal2 := s.bankKeeper.GetBalance(s.ctx, addr, s.denom)
	s.Equal(bal1.Amount.Add(reward), bal2.Amount, "current balance should be equal to previous balance + reward")
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsTwoReporters() {
	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err := s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.denom, reward)))
	s.NoError(err)

	qId := utils.QueryIDFromData(ethQueryData)

	value := []string{"000001", "000002"}
	reporterPower1 := int64(1)
	reporterPower2 := int64(2)
	totalReporterPower := reporterPower1 + reporterPower2

	reporterAddr, err := createReporter(s.ctx, 1, s.reporterkeeper)
	s.NoError(err)
	reporterAddr2, err := createReporter(s.ctx, 2, s.reporterkeeper)
	s.NoError(err)

	// generate 2 reports for ethQueryData
	reports := testutil.GenerateReports([]sdk.AccAddress{reporterAddr, reporterAddr2}, value, []int64{reporterPower1, reporterPower2}, qId)

	testCases := []struct {
		name                 string
		reporterIndex        int
		beforeBalance        sdk.Coin
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with 1 voting power",
			reporterIndex:        0,
			beforeBalance:        s.bankKeeper.GetBalance(s.ctx, reporterAddr, s.denom),
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower1, 1, totalReporterPower, reward),
			delegator:            reporterAddr,
		},
		{
			name:                 "reporter with 2 voting power",
			reporterIndex:        1,
			beforeBalance:        s.bankKeeper.GetBalance(s.ctx, reporterAddr2, s.denom),
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower2, 1, totalReporterPower, reward),
			delegator:            reporterAddr2,
		},
	}
	_, err = s.oraclekeeper.WeightedMedian(s.ctx, reports)
	s.NoError(err)

	res, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	s.NoError(err, "error getting aggregated report")
	tbr, err := s.oraclekeeper.GetTimeBasedRewards(s.ctx, &types.QueryGetTimeBasedRewardsRequest{})
	s.NoError(err, "error getting time based rewards")
	err = s.oraclekeeper.AllocateRewards(s.ctx, res.Report.Reporters, tbr.Reward.Amount, false)
	s.NoError(err, "error allocating rewards")

	// advance height
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.reporterkeeper.WithdrawDelegationRewards(s.ctx, tc.delegator.Bytes(), tc.delegator)
			afterBalance := s.bankKeeper.GetBalance(s.ctx, tc.delegator, s.denom)
			s.Equal(tc.beforeBalance.Amount.Add(tc.afterBalanceIncrease), afterBalance.Amount)
		})
	}
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsThreeReporters() {
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err := s.bankKeeper.SendCoinsFromAccountToModule(s.ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.denom, reward)))
	s.NoError(err)

	values := []string{"000001", "000002", "000003", "000004"}

	reporterPower1 := int64(1)
	reporterPower2 := int64(2)
	reporterPower3 := int64(3)
	totalPower := reporterPower1 + reporterPower2 + reporterPower3

	reporterAddr, err := createReporter(s.ctx, 1, s.reporterkeeper)
	s.NoError(err)
	reporterAddr2, err := createReporter(s.ctx, 2, s.reporterkeeper)
	s.NoError(err)
	reporterAddr3, err := createReporter(s.ctx, 3, s.reporterkeeper)
	s.NoError(err)

	// generate 4 reports for ethQueryData
	qId := utils.QueryIDFromData(ethQueryData)
	reports := testutil.GenerateReports([]sdk.AccAddress{reporterAddr, reporterAddr2, reporterAddr3}, values, []int64{reporterPower1, reporterPower2, reporterPower3}, qId)

	testCases := []struct {
		name                 string
		reporterIndex        int
		beforeBalance        sdk.Coin
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with 100 voting power",
			reporterIndex:        0,
			beforeBalance:        s.bankKeeper.GetBalance(s.ctx, reporterAddr, s.denom),
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower1, 1, totalPower, reward),
			delegator:            reporterAddr,
		},
		{
			name:                 "reporter with 200 voting power",
			reporterIndex:        1,
			beforeBalance:        s.bankKeeper.GetBalance(s.ctx, reporterAddr2, s.denom),
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower2, 1, totalPower, reward),
			delegator:            reporterAddr2,
		},
		{
			name:                 "reporter with 300 voting power",
			reporterIndex:        2,
			beforeBalance:        s.bankKeeper.GetBalance(s.ctx, reporterAddr3, s.denom),
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower3, 1, totalPower, reward),
			delegator:            reporterAddr3,
		},
	}
	s.oraclekeeper.WeightedMedian(s.ctx, reports[:3])

	res, _ := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	tbr, _ := s.oraclekeeper.GetTimeBasedRewards(s.ctx, &types.QueryGetTimeBasedRewardsRequest{})
	err = s.oraclekeeper.AllocateRewards(s.ctx, res.Report.Reporters, tbr.Reward.Amount, false)
	s.NoError(err)
	// advance height
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.reporterkeeper.WithdrawDelegationRewards(s.ctx, tc.delegator.Bytes(), tc.delegator)
			afterBalance := s.bankKeeper.GetBalance(s.ctx, tc.delegator, s.denom)
			expectedAfterBalance := tc.beforeBalance.Amount.Add(tc.afterBalanceIncrease)
			tolerance := expectedAfterBalance.SubRaw(1) //due to rounding int
			withinTolerance := expectedAfterBalance.Equal(afterBalance.Amount) || tolerance.Equal(afterBalance.Amount)
			s.True(withinTolerance)
		})
	}
}

func (s *IntegrationTestSuite) TestCommitQueryMixed() {
	tipper := s.newKeysWithTokens()
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)
	// accs, _, _ := s.createValidatorAccs([]int64{100, 200, 300, 400, 500})
	queryData1, err := s.oraclekeeper.GetCurrentQueryInCycleList(s.ctx)
	s.Nil(err)
	queryData2, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryData3, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData2,
		Amount:    sdk.NewCoin(s.denom, math.NewInt(1000)),
	}
	_, err = msgServer.Tip(s.ctx, &msg)
	s.Nil(err)
	reporterAddr, err := createReporter(s.ctx, 1, s.reporterkeeper)
	s.Nil(err)
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
	salt, err := oracleutils.Salt(32)
	s.Nil(err)
	hash := oracleutils.CalculateCommitment(value, salt)
	s.Nil(err)
	// commit report with query data in cycle list
	commit, _ := report(reporterAddr.String(), value, salt, hash, queryData1)
	_, err = msgServer.CommitReport(s.ctx, &commit)
	s.Nil(err)
	// commit report with query data not in cycle list but has a tip
	commit, _ = report(reporterAddr.String(), value, salt, hash, queryData2)
	_, err = msgServer.CommitReport(s.ctx, &commit)
	s.Nil(err)
	// commit report with query data not in cycle list and has no tip
	commit, _ = report(reporterAddr.String(), value, salt, hash, queryData3)
	_, err = msgServer.CommitReport(s.ctx, &commit)
	s.ErrorContains(err, "query not part of cyclelist")
}

// test tipping a query id not in cycle list and observe the reporters' delegators stake increase in staking module
func (s *IntegrationTestSuite) TestTipQueryNotInCycleListSingleDelegator() {
	require := s.Require()
	s.ctx = s.ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)
	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})
	repAccs := s.CreateAccountsWithTokens(2, 100*1e6)

	stakeAmount := math.NewInt(100 * 1e6)
	tipAmount := math.NewInt(1000)

	tipper := repAccs[0]
	repAcc := repAccs[1]
	valAddr := valAddrs[0]
	delegators := repAccs[1:]

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	// tip. Using msgServer.Tip to handle the transfers and burning of tokens
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.denom, tipAmount),
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.Nil(err)

	// create createReporterStakedWithValidator handles the delegation and staking plus the reporter creation
	commission := reportertypes.NewCommissionWithTime(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec(), s.ctx.BlockTime())
	_, err = createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	delBefore, err := s.stakingKeeper.Delegation(s.ctx, repAcc.Bytes(), valAddr)
	s.Nil(err)
	s.True(delBefore.GetShares().Equal(math.LegacyNewDecFromInt(stakeAmount)), "delegation shares should be equal to the stake amount")

	reporterPower := int64(1)
	value := []string{"000001"}
	reports := testutil.GenerateReports(repAccs[1:], value, []int64{reporterPower}, queryId)
	query, err := s.oraclekeeper.Query.Get(s.ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.Nil(s.oraclekeeper.Query.Set(s.ctx, queryId, query))
	err = s.oraclekeeper.Reports.Set(s.ctx, collections.Join3(queryId, repAcc.Bytes(), query.Id), reports[0])
	s.Nil(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second * 7)) // bypassing offset that expires time to commit/reveal
	err = s.oraclekeeper.SetAggregatedReport(s.ctx)
	s.Nil(err)

	// check that tip is in escrow
	escrowAcct := s.accountKeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.bankKeeper.GetBalance(s.ctx, escrowAcct, s.denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// create reporterMsgServer
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	// withdraw tip
	_, err = reporterMsgServer.WithdrawTip(s.ctx, &reportertypes.MsgWithdrawTip{DelegatorAddress: repAcc.String(), ValidatorAddress: valAddr.String()})
	require.NoError(err)

	// delegation shares should increase after reporting and escrow balance should go back to 0
	delAfter, err := s.stakingKeeper.Delegation(s.ctx, repAcc.Bytes(), valAddr)
	s.Nil(err)
	s.True(delAfter.GetShares().Equal(delBefore.GetShares().Add(math.LegacyNewDec(980))), "delegation shares plus the tip added") // 1000 - 2% tip
	escrowBalance = s.bankKeeper.GetBalance(s.ctx, escrowAcct, s.denom)
	s.True(escrowBalance.IsZero())
}

func (s *IntegrationTestSuite) TestTipQueryNotInCycleListTwoDelegators() {
	require := s.Require()
	msgServer := keeper.NewMsgServerImpl(s.oraclekeeper)
	_, valAddrs, _ := s.createValidatorAccs([]int64{1000})
	accs := s.CreateAccountsWithTokens(3, 100*1e6)

	stakeAmount := math.NewInt(100 * 1e6)
	tipAmount := math.NewInt(1000)

	tipper := accs[0]
	repAcc := accs[1]
	valAddr := valAddrs[0]
	delegators := accs[1:]
	delegator1 := accs[1]
	delegator2 := accs[2]

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	// tip. Using msgServer.Tip to handle the transfers and burning of tokens
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.denom, tipAmount),
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.Nil(err)

	// create createReporterStakedWithValidator handles the delegation and staking plus the reporter creation with 50 percent commission
	commission := reportertypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(5, 1), s.ctx.BlockTime())
	_, err = createReporterStakedWithValidator(s.ctx, s.reporterkeeper, s.stakingKeeper, valAddr, delegators, commission, stakeAmount)
	s.Nil(err)
	source := reportertypes.TokenOrigin{ValidatorAddress: valAddr.String(), Amount: stakeAmount}
	err = DelegateToReporterSingleValidator(s.ctx, s.reporterkeeper, repAcc, delegator2, valAddr, []*reportertypes.TokenOrigin{&source}, stakeAmount)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	del1Before, err := s.stakingKeeper.Delegation(s.ctx, delegator1.Bytes(), valAddr)
	s.Nil(err)
	s.True(del1Before.GetShares().Equal(math.LegacyNewDecFromInt(stakeAmount)), "delegation 1 shares should be equal to the stake amount")

	del2Before, err := s.stakingKeeper.Delegation(s.ctx, delegator2.Bytes(), valAddr)
	s.Nil(err)
	s.True(del2Before.GetShares().Equal(math.LegacyNewDecFromInt(stakeAmount)), "delegation 2 shares should be equal to the stake amount")

	reporterPower := int64(2) // normalize by sdk.DefaultPowerReduction
	value := []string{"000001"}
	reports := testutil.GenerateReports([]sdk.AccAddress{repAcc}, value, []int64{reporterPower}, queryId)
	query, err := s.oraclekeeper.Query.Get(s.ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.oraclekeeper.Query.Set(s.ctx, queryId, query)
	err = s.oraclekeeper.Reports.Set(s.ctx, collections.Join3(queryId, repAcc.Bytes(), query.Id), reports[0])
	s.Nil(err)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Second * 7)) // bypassing offset that expires time to commit/reveal
	err = s.oraclekeeper.SetAggregatedReport(s.ctx)
	s.Nil(err)

	// check tip escrow account
	escrowAcct := s.accountKeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.bankKeeper.GetBalance(s.ctx, escrowAcct, s.denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// withdraw self delegation from tip escrow
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	_, err = reporterMsgServer.WithdrawTip(s.ctx, &reportertypes.MsgWithdrawTip{DelegatorAddress: delegator1.String(), ValidatorAddress: valAddr.String()})
	require.NoError(err)

	// delegation shares should increase after reporting and withdrawing
	del1After, err := s.stakingKeeper.Delegation(s.ctx, delegator1.Bytes(), valAddr)
	s.Nil(err)
	// 980 = 1000 - 2% tip, 980 / 2 = 490 for each delegator but with 50 percent commission for the reporter would be 490 + (490 / 2) = 735
	s.True(del1After.GetShares().Equal(del1Before.GetShares().Add(math.LegacyNewDec(735))), "delegation 1 (self delegation) shares should be half the tip plus 50 percent commission")

	// withdraw del2 delegation from tip escrow
	_, err = reporterMsgServer.WithdrawTip(s.ctx, &reportertypes.MsgWithdrawTip{DelegatorAddress: delegator2.String(), ValidatorAddress: valAddr.String()})
	require.NoError(err)

	// 980 - 735 = 245
	del2After, err := s.stakingKeeper.Delegation(s.ctx, delegator2.Bytes(), valAddr)
	s.Nil(err)
	s.True(del2After.GetShares().Equal(del2Before.GetShares().Add(math.LegacyNewDec(245))), "delegation 2 shares should be half the tip minus 50 percent reporter commission")
}
