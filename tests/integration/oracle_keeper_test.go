package integration_test

import (
	"encoding/hex"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *IntegrationTestSuite) TestTipping() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	userTips, err := s.Setup.Oraclekeeper.GetUserTips(s.Setup.Ctx, addr)
	s.NoError(err)
	s.Equal(userTips.Int64(), tips.Int64())

	// tip same query again
	_, err = msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
	s.NoError(err)
	// tips should be 2x
	s.Equal(tip.Sub(twoPercent).Amount.Mul(math.NewInt(2)), tips)

	// total tips overall
	userTips, err = s.Setup.Oraclekeeper.GetUserTips(s.Setup.Ctx, addr)
	s.NoError(err)
	s.Equal(userTips, tips)

	// tip different query
	btcQueryId := utils.QueryIDFromData(btcQueryData)

	_, err = msgServer.Tip(s.Setup.Ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	s.NoError(err)
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, btcQueryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	userQueryTips, _ := s.Setup.Oraclekeeper.Tips.Get(s.Setup.Ctx, collections.Join(btcQueryId, addr.Bytes()))
	s.Equal(userQueryTips, tips)
	userTips, err = s.Setup.Oraclekeeper.GetUserTips(s.Setup.Ctx, addr)
	s.NoError(err)
	s.Equal(userTips, tips.Add(tips).Add(tips))
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)

	// Get current tip
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	resp, err := queryServer.GetCurrentTip(s.Setup.Ctx, &types.QueryGetCurrentTipRequest{QueryData: hex.EncodeToString(ethQueryData)})
	s.NoError(err)
	s.Equal(tip.Amount.Sub(twoPercent.Amount), resp.Tips)
}

// test tipping, reporting and allocation of rewards
func (s *IntegrationTestSuite) TestTippingReporting() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))

	value := testutil.EncodeValue(29266)
	reveal := report(repAccs[0].String(), value, ethQueryData)
	_, err = msgServer.SubmitValue(s.Setup.Ctx, &reveal)
	s.Nil(err)
	// advance time to expire the query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx)
	s.Nil(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(queryId)})
	s.Nil(err)
	s.Equal(res.Aggregate.AggregateReporter, repAccs[0].String())
	// tip should be 0 after aggregated report
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(s.Setup.Ctx, queryId)
	s.Nil(err)
	s.Equal(tips, math.ZeroInt())
	totalTips, err := s.Setup.Oraclekeeper.GetTotalTips(s.Setup.Ctx)
	s.Nil(err)
	s.Equal(totalTips, tip.Sub(twoPercent).Amount) // total tips should be equal to the tipped amount minus 2% burned
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := math.NewInt(1000)
	twoPercent := tip.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, tip),
	}
	_, err := msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)

	// Get current tip
	resp, err := queryServer.GetUserTipTotal(s.Setup.Ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(resp.TotalTips, tip.Sub(twoPercent))
	// Check total tips without a given query data
	respUserTotal, err := queryServer.GetUserTipTotal(s.Setup.Ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(respUserTotal.TotalTips, tip.Sub(twoPercent))
}

func (s *IntegrationTestSuite) TestSmallTip() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)

	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(10))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	accBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, addr, s.Setup.Denom)
	modBalanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, authtypes.NewModuleAddress(types.ModuleName), s.Setup.Denom)
	_, err := msgServer.Tip(s.Setup.Ctx, &msg)
	s.NoError(err)
	accBalanceAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, addr, s.Setup.Denom)
	modBalanceAfter := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, authtypes.NewModuleAddress(types.ModuleName), s.Setup.Denom)
	s.Equal(accBalanceBefore.Amount.Sub(tip.Amount), accBalanceAfter.Amount)
	s.Equal(modBalanceBefore.Amount.Add(tip.Amount).Sub(twoPercent.Amount), modBalanceAfter.Amount)
}

func (s *IntegrationTestSuite) TestMedianReports() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200, 300, 400, 500})
	tipper := s.newKeysWithTokens()
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, rep, []byte{})
		s.NoError(err)
	}
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
			value:         testutil.EncodeValue(162926),
			stakeAmount:   math.NewInt(1_000_000),
			power:         1,
		},
		{
			name:          "reporter 2",
			reporterIndex: 1,
			value:         testutil.EncodeValue(362926),
			stakeAmount:   math.NewInt(2_000_000),
			power:         2,
		},
		{
			name:          "reporter 3",
			reporterIndex: 2,
			value:         testutil.EncodeValue(262926),
			stakeAmount:   math.NewInt(3_000_000),
			power:         3,
		},
		{
			name:          "reporter 4",
			reporterIndex: 3,
			value:         testutil.EncodeValue(562926),
			stakeAmount:   math.NewInt(4_000_000),
			power:         4,
		},
		{
			name:          "reporter 5",
			reporterIndex: 4,
			value:         testutil.EncodeValue(462926),
			stakeAmount:   math.NewInt(5_000_000),
			power:         5,
		},
	}
	_, err := msgServer.Tip(s.Setup.Ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(1000))})
	s.Nil(err)
	addr := make([]sdk.AccAddress, len(reporters))
	for i, r := range reporters {
		s.T().Run(r.name, func(t *testing.T) {
			// create reporter
			addr[r.reporterIndex] = repAccs[i]
			reveal := report(repAccs[i].String(), r.value, ethQueryData)
			_, err = msgServer.SubmitValue(s.Setup.Ctx, &reveal)
			s.Nil(err)
		})
	}
	// advance time to expire query and aggregate report
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 3) // bypass time to expire query so it can be aggregated

	_, _ = s.Setup.App.EndBlocker(s.Setup.Ctx)
	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx))
	// check median
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	expectedMedianReporterIndex := 4
	expectedMedianReporter := addr[expectedMedianReporterIndex].String()
	s.Equal(expectedMedianReporter, res.Aggregate.AggregateReporter)
	s.Equal(reporters[expectedMedianReporterIndex].value, res.Aggregate.AggregateValue)
}

func report(creator, value string, qdata []byte) types.MsgSubmitValue {
	reveal := types.MsgSubmitValue{
		Creator:   creator,
		QueryData: qdata,
		Value:     value,
	}
	return reveal
}

func (s *IntegrationTestSuite) TestGetCylceListQueries() {
	accs, _, _ := s.createValidatorAccs([]uint64{100, 200, 300, 400, 500})
	// Get supported queries
	resp, err := s.Setup.Oraclekeeper.GetCyclelist(s.Setup.Ctx)
	s.NoError(err)
	s.Equal(resp, [][]byte{trbQueryData, ethQueryData, btcQueryData})

	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msgContent := &types.MsgUpdateCyclelist{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Cyclelist: [][]byte{matic},
	}
	proposal1, err := s.Setup.Govkeeper.SubmitProposal(s.Setup.Ctx, []sdk.Msg{msgContent}, "", "test", "description", accs[0], false)
	s.NoError(err)

	govParams, err := s.Setup.Govkeeper.Params.Get(s.Setup.Ctx)
	s.NoError(err)
	votingStarted, err := s.Setup.Govkeeper.AddDeposit(s.Setup.Ctx, proposal1.Id, accs[0], govParams.MinDeposit)
	s.NoError(err)
	s.True(votingStarted)
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusVotingPeriod)
	err = s.Setup.Govkeeper.AddVote(s.Setup.Ctx, proposal1.Id, accs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.Setup.Govkeeper.AddVote(s.Setup.Ctx, proposal1.Id, accs[1], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.Setup.Govkeeper.AddVote(s.Setup.Ctx, proposal1.Id, accs[2], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal1.Id)
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(s.Setup.Ctx.BlockTime().Add(time.Hour * 24 * 2))
	s.NoError(gov.EndBlocker(s.Setup.Ctx, s.Setup.Govkeeper))
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(s.Setup.Ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusPassed)
	resp, err = s.Setup.Oraclekeeper.GetCyclelist(s.Setup.Ctx)
	s.NoError(err)
	s.Equal([][]byte{matic}, resp)
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsOneReporter() {
	reporterPower := uint64(1)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{reporterPower})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	qId := utils.QueryIDFromData(ethQueryData)
	stake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0], qId)
	s.NoError(err)

	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)

	// testing for a query id and check if the reporter gets the reward, bypassing the commit/reveal process
	value := []string{"000001"}
	reports := testutil.GenerateReports([]sdk.AccAddress{repAccs[0]}, value, []uint64{reporterPower}, qId)

	_, err = s.Setup.Oraclekeeper.WeightedMedian(s.Setup.Ctx, reports[:1], 1)
	s.NoError(err)
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.NoError(err)
	s.Equal(res.Aggregate.AggregateReportIndex, uint64(0), "single report should be at index 0")

	tbr, err := queryServer.GetTimeBasedRewards(s.Setup.Ctx, &types.QueryGetTimeBasedRewardsRequest{})
	s.NoError(err)

	err = s.Setup.Oraclekeeper.AllocateRewards(s.Setup.Ctx, []*types.Aggregate{res.Aggregate}, tbr.Reward.Amount, minttypes.TimeBasedRewards)
	s.NoError(err)
	// advance height
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)

	tip, err := s.Setup.Reporterkeeper.SelectorTips.Get(s.Setup.Ctx, repAccs[0].Bytes())
	s.NoError(err)
	s.Equal(tip.TruncateInt(), reward, "reporter should get the reward")
	// withdraw the reward
	repServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err = repServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: repAccs[0].String(), ValidatorAddress: valAddrs[0].String()})
	s.NoError(err)
	bond, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, repAccs[0])
	s.NoError(err)
	s.Equal(stake.Add(reward), bond, "current balance should be equal to previous balance + reward")
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsTwoReporters() {
	qId := utils.QueryIDFromData(ethQueryData)

	value := []string{"000001", "000002"}
	reporterPower1 := uint64(1)
	reporterPower2 := uint64(2)
	totalReporterPower := reporterPower1 + reporterPower2
	repAccs, _, _ := s.createValidatorAccs([]uint64{reporterPower1, reporterPower2})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0], qId)
	s.NoError(err)
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[1], qId)
	s.NoError(err)
	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)
	// generate 2 reports for ethQueryData
	reports := testutil.GenerateReports([]sdk.AccAddress{repAccs[0], repAccs[1]}, value, []uint64{reporterPower1, reporterPower2}, qId)

	testCases := []struct {
		name                 string
		reporterIndex        int
		beforeBalance        math.Int
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with 1 voting power",
			reporterIndex:        0,
			beforeBalance:        reporterStake,
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower1, 1, totalReporterPower, reward).TruncateInt(),
			delegator:            repAccs[0],
		},
		{
			name:                 "reporter with 2 voting power",
			reporterIndex:        1,
			beforeBalance:        reporterStake2,
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower2, 1, totalReporterPower, reward).TruncateInt(),
			delegator:            repAccs[1],
		},
	}
	_, err = s.Setup.Oraclekeeper.WeightedMedian(s.Setup.Ctx, reports, 1)
	s.NoError(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.NoError(err, "error getting aggregated report")
	tbr, err := queryServer.GetTimeBasedRewards(s.Setup.Ctx, &types.QueryGetTimeBasedRewardsRequest{})
	s.NoError(err, "error getting time based rewards")
	err = s.Setup.Oraclekeeper.AllocateRewards(s.Setup.Ctx, []*types.Aggregate{res.Aggregate}, tbr.Reward.Amount, minttypes.TimeBasedRewards)
	s.NoError(err, "error allocating rewards")
	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	// advance height
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, err = reporterServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: tc.delegator.String(), ValidatorAddress: sdk.ValAddress(tc.delegator).String()})
			s.NoError(err)
			afterBalance, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, tc.delegator)
			s.NoError(err)
			s.Equal(tc.beforeBalance.Add(tc.afterBalanceIncrease), afterBalance)
		})
	}
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsThreeReporters() {
	qId := utils.QueryIDFromData(ethQueryData)
	values := []string{"000001", "000002", "000003", "000004"}

	reporterPower1 := uint64(1)
	reporterPower2 := uint64(2)
	reporterPower3 := uint64(3)
	totalPower := reporterPower1 + reporterPower2 + reporterPower3
	repAccs, _, _ := s.createValidatorAccs([]uint64{reporterPower1, reporterPower2, reporterPower3})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[2], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[2], reportertypes.NewSelection(repAccs[2], 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0], qId)
	s.NoError(err)
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[1], qId)
	s.NoError(err)
	reporterStake3, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[2], qId)
	s.NoError(err)

	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)
	// generate 4 reports for ethQueryData
	reports := testutil.GenerateReports([]sdk.AccAddress{repAccs[0], repAccs[1], repAccs[2]}, values, []uint64{reporterPower1, reporterPower2, reporterPower3}, qId)

	testCases := []struct {
		name                 string
		reporterIndex        int
		beforeBalance        math.Int
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with 100 voting power",
			reporterIndex:        0,
			beforeBalance:        reporterStake,
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower1, 1, totalPower, reward).TruncateInt(),
			delegator:            repAccs[0],
		},
		{
			name:                 "reporter with 200 voting power",
			reporterIndex:        1,
			beforeBalance:        reporterStake2,
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower2, 1, totalPower, reward).TruncateInt(),
			delegator:            repAccs[1],
		},
		{
			name:                 "reporter with 300 voting power",
			reporterIndex:        2,
			beforeBalance:        reporterStake3,
			afterBalanceIncrease: keeper.CalculateRewardAmount(reporterPower3, 1, totalPower, reward).TruncateInt(),
			delegator:            repAccs[2],
		},
	}
	_, err = s.Setup.Oraclekeeper.WeightedMedian(s.Setup.Ctx, reports[:3], 1)
	s.NoError(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, _ := queryServer.GetCurrentAggregateReport(s.Setup.Ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	tbr, _ := queryServer.GetTimeBasedRewards(s.Setup.Ctx, &types.QueryGetTimeBasedRewardsRequest{})
	err = s.Setup.Oraclekeeper.AllocateRewards(s.Setup.Ctx, []*types.Aggregate{res.Aggregate}, tbr.Reward.Amount, minttypes.TimeBasedRewards)
	s.NoError(err)
	// advance height
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, err = reporterServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: tc.delegator.String(), ValidatorAddress: sdk.ValAddress(tc.delegator).String()})
			s.NoError(err)
			afterBalance, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, tc.delegator)
			s.NoError(err)
			expectedAfterBalance := tc.beforeBalance.Add(tc.afterBalanceIncrease)
			tolerance := expectedAfterBalance.SubRaw(1)
			withinTolerance := expectedAfterBalance.Equal(afterBalance) || tolerance.Equal(afterBalance)
			s.True(withinTolerance)
		})
	}
}

func (s *IntegrationTestSuite) TestTokenBridgeQuery() {
	repAccs, _, _ := s.Setup.CreateValidators(5)
	ok := s.Setup.Oraclekeeper
	ctx := s.Setup.Ctx
	app := s.Setup.App
	msgServer := keeper.NewMsgServerImpl(ok)
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
	}
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	querydata, err := spec.EncodeData("TRBBridge", `["true","1"]`)
	s.NoError(err)

	reporter1, reporter2, reporter3, reporter4, reporter5 := repAccs[0], repAccs[1], repAccs[2], repAccs[3], repAccs[4]
	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	msgSubmitValue := types.MsgSubmitValue{
		Creator:   reporter1.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 3, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter2.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter3.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)

	_, _ = app.EndBlocker(ctx)

	// should be exactly 1h mark
	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	agg, _, err := ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.Error(err)
	s.Nil(agg)

	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter4.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	// time plus offset
	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 2000, Time: ctx.BlockTime().Add(time.Second * 11)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	agg, _, err = ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.NoError(err)
	s.Equal(len(agg.Reporters), 4)

	// new report that starts a new cycle
	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter5.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 2000, Time: ctx.BlockTime().Add(time.Hour + time.Second*11)})

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	agg, _, err = ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.NoError(err)
	s.Equal(len(agg.Reporters), 1)
}

func (s *IntegrationTestSuite) TestTokenBridgeQueryDirectreveal() {
	repAccs, _, _ := s.Setup.CreateValidators(5)
	ok := s.Setup.Oraclekeeper
	ctx := s.Setup.Ctx
	app := s.Setup.App
	msgServer := keeper.NewMsgServerImpl(ok)
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
	}
	spec := registrytypes.DataSpec{
		AbiComponents: []*registrytypes.ABIComponent{
			{Name: "tolayer", FieldType: "bool"},
			{Name: "depositId", FieldType: "uint256"},
		},
	}
	querydata, err := spec.EncodeData("TRBBridge", `["true","1"]`)
	s.NoError(err)

	reporter1, reporter2, reporter3, reporter4, reporter5 := repAccs[0], repAccs[1], repAccs[2], repAccs[3], repAccs[4]
	value := "000000000000000000000000000000000000000000000058528649cf90ee0000"

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	msgSubmitValue := types.MsgSubmitValue{
		Creator:   reporter1.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter2.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter3.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)

	_, _ = app.EndBlocker(ctx)

	// should be exactly 1h mark
	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 1, Time: ctx.BlockTime().Add(time.Minute * 20)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	agg, _, err := ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.Error(err)
	s.Nil(agg)

	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter4.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	// time plus offset
	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 2000, Time: ctx.BlockTime().Add(time.Second * 11)})
	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)
	agg, _, err = ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.NoError(err)
	s.Equal(len(agg.Reporters), 4)

	// new report that starts a new cycle
	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter5.String(),
		QueryData: querydata,
		Value:     value,
	}
	_, err = msgServer.SubmitValue(ctx, &msgSubmitValue)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	ctx = ctx.WithBlockHeader(cmtproto.Header{Height: ctx.BlockHeight() + 2000, Time: ctx.BlockTime().Add(time.Hour + time.Second*11)})

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	agg, _, err = ok.GetCurrentAggregateReport(ctx, crypto.Keccak256(querydata))
	s.NoError(err)
	s.Equal(len(agg.Reporters), 1)
}

// func (s *IntegrationTestSuite) TestCommitQueryMixed() {
// 	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
// 	repAccs, _, _ := s.createValidatorAccs([]uint64{100})
// 	s.NoError(s.Setup.Oraclekeeper.RotateQueries(s.Setup.Ctx))
// 	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
// 	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))
// 	_, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0])
// 	s.NoError(err)
// 	queryData1, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
// 	s.Nil(err)

// 	queryData2, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
// 	queryData3, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000005737465746800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")

// 	tipper := s.newKeysWithTokens()
// 	msg := types.MsgTip{
// 		Tipper:    tipper.String(),
// 		QueryData: queryData2,
// 		Amount:    sdk.NewCoin(s.Setup.Denom, math.NewInt(1000)),
// 	}
// 	_, err = msgServer.Tip(s.Setup.Ctx, &msg)
// 	s.Nil(err)

// 	userTips, err := s.Setup.Oraclekeeper.GetUserTips(s.Setup.Ctx, tipper)
// 	s.Nil(err)
// 	// tip should be 1000 minus 2% burned
// 	s.Equal(userTips, math.NewInt(980))
// 	value := "000000000000000000000000000000000000000000000058528649cf0ee0000"
// 	salt, err := oracleutils.Salt(32)
// 	s.Nil(err)
// 	hash := oracleutils.CalculateCommitment(value, salt)
// 	s.Nil(err)
// 	// commit report with query data in cycle list
// 	commit, _ := report(repAccs[0].String(), value, salt, hash, queryData1)
// 	_, err = msgServer.CommitReport(s.Setup.Ctx, &commit)
// 	s.Nil(err)
// 	// commit report with query data not in cycle list but has a tip
// 	commit, _ = report(repAccs[0].String(), value, salt, hash, queryData2)
// 	_, err = msgServer.CommitReport(s.Setup.Ctx, &commit)
// 	s.Nil(err)
// 	// commit report with query data not in cycle list and has no tip
// 	commit, _ = report(repAccs[0].String(), value, salt, hash, queryData3)
// 	_, err = msgServer.CommitReport(s.Setup.Ctx, &commit)
// 	s.ErrorContains(err, "query doesn't exist plus not a bridge deposit: not a token deposit")
// }

// test tipping a query id not in cycle list and observe the reporters' delegators stake increase in staking module
func (s *IntegrationTestSuite) TestTipQueryNotInCycleListSingleDelegator() {
	require := s.Require()
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{1000})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	stakeAmount, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0], queryId)
	require.NoError(err)
	tipAmount := math.NewInt(1000)

	tipper := s.newKeysWithTokens()

	valAddr := valAddrs[0]

	// tip. Using msgServer.Tip to handle the transfers and burning of tokens
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, tipAmount),
	}
	_, err = msgServer.Tip(s.Setup.Ctx, &msg)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	delBefore, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, repAccs[0].Bytes(), valAddr)
	s.Nil(err)
	s.True(delBefore.GetShares().Equal(math.LegacyNewDecFromInt(stakeAmount)), "delegation shares should be equal to the stake amount")

	reporterPower := uint64(1)
	value := []string{"000001"}
	reports := testutil.GenerateReports(repAccs, value, []uint64{reporterPower}, queryId)
	query, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.Nil(s.Setup.Oraclekeeper.Query.Set(s.Setup.Ctx, collections.Join(queryId, query.Id), query))
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(queryId, repAccs[0].Bytes(), query.Id), reports[0])
	s.Nil(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx)
	s.Nil(err)

	// check that tip is in escrow
	escrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, escrowAcct, s.Setup.Denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// create reporterMsgServer
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	// withdraw tip
	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: repAccs[0].String(), ValidatorAddress: valAddr.String()})
	require.NoError(err)

	// delegation shares should increase after reporting and escrow balance should go back to 0
	delAfter, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, repAccs[0].Bytes(), valAddr)
	s.Nil(err)
	s.True(delAfter.GetShares().Equal(delBefore.GetShares().Add(math.LegacyNewDec(980))), "delegation shares plus the tip added") // 1000 - 2% tip
	escrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, escrowAcct, s.Setup.Denom)
	s.True(escrowBalance.IsZero())
}

func (s *IntegrationTestSuite) TestTipQueryNotInCycleListTwoDelegators() {
	require := s.Require()
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{1, 2})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	reporterStake1, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[0], queryId)
	require.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt())))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(s.Setup.Ctx, repAccs[1], queryId)
	require.NoError(err)

	tipAmount := math.NewInt(1000)

	tipper := s.newKeysWithTokens()
	valAddr1 := valAddrs[0]
	valAddr2 := valAddrs[1]
	delegator1 := repAccs[0]
	delegator2 := repAccs[1]

	// tip. Using msgServer.Tip to handle the transfers and burning of tokens
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, tipAmount),
	}
	_, err = msgServer.Tip(s.Setup.Ctx, &msg)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	del1Before, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, delegator1.Bytes(), valAddr1)
	s.Nil(err)
	s.True(del1Before.GetShares().Equal(math.LegacyNewDecFromInt(reporterStake1)), "delegation 1 shares should be equal to the stake amount")

	del2Before, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, delegator2.Bytes(), valAddr2)
	s.Nil(err)
	s.True(del2Before.GetShares().Equal(math.LegacyNewDecFromInt(reporterStake2)), "delegation 2 shares should be equal to the stake amount")

	reporterPower := uint64(1)
	reporterPower2 := uint64(2)
	value := []string{"000001", "000002"}
	reports := testutil.GenerateReports([]sdk.AccAddress{repAccs[0], repAccs[1]}, value, []uint64{reporterPower, reporterPower2}, queryId)
	query, err := s.Setup.Oraclekeeper.CurrentQuery(s.Setup.Ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.NoError(s.Setup.Oraclekeeper.Query.Set(s.Setup.Ctx, collections.Join(queryId, query.Id), query))
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(queryId, repAccs[0].Bytes(), query.Id), reports[0])
	s.Nil(err)
	err = s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(queryId, repAccs[1].Bytes(), query.Id), reports[1])
	s.Nil(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(s.Setup.Ctx)
	s.Nil(err)

	// check tip escrow account
	escrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, escrowAcct, s.Setup.Denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// withdraw self delegation from tip escrow
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: delegator1.String(), ValidatorAddress: valAddr1.String()})
	require.NoError(err)

	// delegation shares should increase after reporting and withdrawing
	del1After, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, delegator1.Bytes(), valAddr1)
	s.Nil(err)
	s.True(del1After.GetShares().Equal(del1Before.GetShares().Add(math.LegacyNewDec(326))), "delegation 1 (self delegation) shares should be half the tip plus 50 percent commission")
	// withdraw del2 delegation from tip escrow
	_, err = reporterMsgServer.WithdrawTip(s.Setup.Ctx, &reportertypes.MsgWithdrawTip{SelectorAddress: delegator2.String(), ValidatorAddress: valAddr2.String()})
	require.NoError(err)

	del2After, err := s.Setup.Stakingkeeper.Delegation(s.Setup.Ctx, delegator2.Bytes(), valAddr2)
	s.Nil(err)
	s.True(del2After.GetShares().Equal(del2Before.GetShares().Add(math.LegacyNewDec(653))), "delegation 2 shares should be half the tip minus 50 percent reporter commission")
}
