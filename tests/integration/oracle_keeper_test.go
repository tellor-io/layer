package integration_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/testutil"
	"github.com/tellor-io/layer/utils"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func (s *IntegrationTestSuite) TestTipping() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	ctx := s.Setup.Ctx
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.Setup.Oraclekeeper.GetQueryTip(ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	userTips, err := s.Setup.Oraclekeeper.GetUserTips(ctx, addr)
	s.NoError(err)
	s.Equal(userTips.Int64(), tips.Int64())

	// tip same query again
	_, err = msgServer.Tip(ctx, &msg)
	s.NoError(err)
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(ctx, queryId)
	s.NoError(err)
	// tips should be 2x
	s.Equal(tip.Sub(twoPercent).Amount.Mul(math.NewInt(2)), tips)

	// total tips overall
	userTips, err = s.Setup.Oraclekeeper.GetUserTips(ctx, addr)
	s.NoError(err)
	s.Equal(userTips, tips)

	// tip different query
	btcQueryId := utils.QueryIDFromData(btcQueryData)

	_, err = msgServer.Tip(ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	s.NoError(err)
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(ctx, btcQueryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	userTips, err = s.Setup.Oraclekeeper.GetUserTips(ctx, addr)
	s.NoError(err)
	s.Equal(userTips, tips.Add(tips).Add(tips))
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	ctx := s.Setup.Ctx
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)

	// Get current tip
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	resp, err := queryServer.GetCurrentTip(ctx, &types.QueryGetCurrentTipRequest{QueryData: hex.EncodeToString(ethQueryData)})
	s.NoError(err)
	s.Equal(tip.Amount.Sub(twoPercent.Amount), resp.Tips)
}

// test tipping, reporting and allocation of rewards
func (s *IntegrationTestSuite) TestTippingReporting() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	ctx := s.Setup.Ctx
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200})
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)

	queryId := utils.QueryIDFromData(ethQueryData)

	tips, err := s.Setup.Oraclekeeper.GetQueryTip(ctx, queryId)
	s.NoError(err)
	s.Equal(tip.Sub(twoPercent).Amount, tips)

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker2")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))

	value := testutil.EncodeValue(29266)
	reveal := report(repAccs[0].String(), value, ethQueryData)
	query, _ := s.Setup.Oraclekeeper.CurrentQuery(ctx, (queryId))
	_, err = msgServer.SubmitValue(ctx, &reveal)
	s.Nil(err)
	// advance time to expire the query and aggregate report
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(ctx)
	s.Nil(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(queryId)})
	s.Nil(err)
	med, _ := s.Setup.Oraclekeeper.AggregateValue.Get(ctx, query.Id)
	fmt.Println(med.Value, res.Aggregate.AggregateValue)
	s.Equal(res.Aggregate.AggregateReporter, repAccs[0].String())
	// tip should be 0 after aggregated report
	tips, err = s.Setup.Oraclekeeper.GetQueryTip(ctx, queryId)
	s.Nil(err)
	s.Equal(tips, math.ZeroInt())
	totalTips, err := s.Setup.Oraclekeeper.GetTotalTips(ctx)
	s.Nil(err)
	s.Equal(totalTips, tip.Sub(twoPercent).Amount) // total tips should be equal to the tipped amount minus 2% burned
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	ctx := s.Setup.Ctx
	addr := s.newKeysWithTokens()

	tip := math.NewInt(100_000)
	twoPercent := tip.Mul(math.NewInt(2)).Quo(math.NewInt(100))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, tip),
	}
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)

	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)

	// Get current tip
	resp, err := queryServer.GetUserTipTotal(ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(resp.TotalTips, tip.Sub(twoPercent))
	// Check total tips without a given query data
	respUserTotal, err := queryServer.GetUserTipTotal(ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(respUserTotal.TotalTips, tip.Sub(twoPercent))
}

func (s *IntegrationTestSuite) TestSmallTip() {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	ctx := s.Setup.Ctx
	addr := s.newKeysWithTokens()

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(10_000))
	twoPercent := sdk.NewCoin(s.Setup.Denom, tip.Amount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	accBalanceBefore := s.Setup.Bankkeeper.GetBalance(ctx, addr, s.Setup.Denom)
	modBalanceBefore := s.Setup.Bankkeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), s.Setup.Denom)
	_, err := msgServer.Tip(ctx, &msg)
	s.NoError(err)
	accBalanceAfter := s.Setup.Bankkeeper.GetBalance(ctx, addr, s.Setup.Denom)
	modBalanceAfter := s.Setup.Bankkeeper.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), s.Setup.Denom)
	s.Equal(accBalanceBefore.Amount.Sub(tip.Amount), accBalanceAfter.Amount)
	s.Equal(modBalanceBefore.Amount.Add(tip.Amount).Sub(twoPercent.Amount), modBalanceAfter.Amount)
}

func (s *IntegrationTestSuite) TestMedianReports() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx := s.Setup.Ctx
	ctx = ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, _, _ := s.createValidatorAccs([]uint64{100, 200, 300, 400, 500})
	tipper := s.newKeysWithTokens()
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
		s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
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
	_, err := msgServer.Tip(s.Setup.Ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))})
	s.Nil(err)
	addr := make([]sdk.AccAddress, len(reporters))
	for i, r := range reporters {
		s.T().Run(r.name, func(t *testing.T) {
			// create reporter
			addr[r.reporterIndex] = repAccs[i]
			reveal := report(repAccs[i].String(), r.value, ethQueryData)
			_, err = msgServer.SubmitValue(ctx, &reveal)
			s.Nil(err)
		})
	}
	// advance time to expire query and aggregate report
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3) // bypass time to expire query so it can be aggregated

	_, _ = s.Setup.App.EndBlocker(ctx)
	s.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))
	// check median
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	res, err := queryServer.GetCurrentAggregateReport(ctx, &types.QueryGetCurrentAggregateReportRequest{QueryId: hex.EncodeToString(qId)})
	s.Nil(err)
	expectedMedianReporterIndex := 4
	expectedMedianReporter := addr[expectedMedianReporterIndex].String()
	s.Equal(expectedMedianReporter, res.Aggregate.AggregateReporter)
	s.Equal(reporters[expectedMedianReporterIndex].value, res.Aggregate.AggregateValue)
	query, _ := s.Setup.Oraclekeeper.CurrentQuery(ctx, qId)
	med, _ := s.Setup.Oraclekeeper.AggregateValue.Get(ctx, query.Id)
	fmt.Println(med.Value, res.Aggregate.AggregateValue)
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
	ctx := s.Setup.Ctx
	accs, _, _ := s.createValidatorAccs([]uint64{100, 200, 300, 400, 500})
	// Get supported queries
	resp, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)
	s.Equal(resp, [][]byte{trbQueryData, ethQueryData, btcQueryData})

	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	msgContent := &types.MsgUpdateCyclelist{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Cyclelist: [][]byte{matic},
	}
	proposal1, err := s.Setup.Govkeeper.SubmitProposal(ctx, []sdk.Msg{msgContent}, "", "test", "description", accs[0], false)
	s.NoError(err)

	govParams, err := s.Setup.Govkeeper.Params.Get(ctx)
	s.NoError(err)
	votingStarted, err := s.Setup.Govkeeper.AddDeposit(ctx, proposal1.Id, accs[0], govParams.MinDeposit)
	s.NoError(err)
	s.True(votingStarted)
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusVotingPeriod)
	err = s.Setup.Govkeeper.AddVote(ctx, proposal1.Id, accs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.Setup.Govkeeper.AddVote(ctx, proposal1.Id, accs[1], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.Setup.Govkeeper.AddVote(ctx, proposal1.Id, accs[2], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(ctx, proposal1.Id)
	s.NoError(err)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour * 24 * 2))
	s.NoError(gov.EndBlocker(ctx, s.Setup.Govkeeper))
	proposal1, err = s.Setup.Govkeeper.Proposals.Get(ctx, proposal1.Id)
	s.NoError(err)
	s.True(proposal1.Status == v1.StatusPassed)
	resp, err = s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)
	s.Equal([][]byte{matic}, resp)
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsOneReporter() {
	ctx := s.Setup.Ctx
	reporterPower := uint64(1)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{reporterPower})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	qId := utils.QueryIDFromData(ethQueryData)
	stake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[0], qId)
	s.NoError(err)

	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)

	// With the new liveness-weighted system, TBR is distributed at the end of a liveness period
	// based on (power × liveness) / totalWeight
	// For a single reporter reporting on all cyclelist queries, they should get 100% of TBR

	// Get the cyclelist to set up query opportunities
	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)

	// Reset liveness data to clear any setup-time increments (RotateQueries runs during FinalizeBlock)
	s.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Increment standard opportunities (simulating one cycle completion)
	s.NoError(s.Setup.Oraclekeeper.IncrementStandardOpportunities(ctx))

	// Set up query opportunities for all cyclelist queries (simulating a full cycle)
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		s.NoError(s.Setup.Oraclekeeper.IncrementQueryOpportunities(ctx, queryId))
	}

	// Track that the reporter reported on all cyclelist queries
	// This simulates the reporter being the only one in each aggregate
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		// UpdateReporterLiveness tracks both the query and accumulates power
		// With per-aggregate power share: reporter is only one in aggregate, so gets 100%
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[0], queryId, reporterPower, reporterPower, false))
	}

	// Distribute liveness rewards - this is what happens at the end of a cycle
	s.NoError(s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx))

	// advance height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	// withdraw the reward
	repServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err = repServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
		SelectorAddress: repAccs[0].String(), ValidatorAddress: valAddrs[0].String(),
	})
	s.NoError(err)
	bond, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, repAccs[0])
	s.NoError(err)
	// Allow tolerance of 1 for rounding (100/3 queries = 33.33... each, total may be 99 or 100)
	expectedBalance := stake.Add(reward)
	tolerance := expectedBalance.SubRaw(1)
	withinTolerance := expectedBalance.Equal(bond) || tolerance.Equal(bond)
	s.True(withinTolerance, "current balance should be within 1 of previous balance + reward")
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsTwoReporters() {
	qId := utils.QueryIDFromData(ethQueryData)
	ctx := s.Setup.Ctx
	reporterPower1 := uint64(1)
	reporterPower2 := uint64(2)
	totalReporterPower := reporterPower1 + reporterPower2
	repAccs, _, _ := s.createValidatorAccs([]uint64{reporterPower1, reporterPower2})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker2")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[0], qId)
	s.NoError(err)
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[1], qId)
	s.NoError(err)

	// send timebasedrewards tokens to oracle module to pay reporters with
	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)

	// With liveness-weighted TBR distribution:
	// reward = (power × liveness) / totalWeight × totalReward
	// If both reporters have 100% liveness, it simplifies to power / totalPower × totalReward
	testCases := []struct {
		name                 string
		beforeBalance        math.Int
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with 1 voting power",
			beforeBalance:        reporterStake,
			afterBalanceIncrease: reporterkeeper.CalculateRewardAmount(reporterPower1, totalReporterPower, reward).TruncateInt(),
			delegator:            repAccs[0],
		},
		{
			name:                 "reporter with 2 voting power",
			beforeBalance:        reporterStake2,
			afterBalanceIncrease: reporterkeeper.CalculateRewardAmount(reporterPower2, totalReporterPower, reward).TruncateInt(),
			delegator:            repAccs[1],
		},
	}

	// Get the cyclelist to set up query opportunities
	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)

	// Reset liveness data to clear any setup-time increments (RotateQueries runs during FinalizeBlock)
	s.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Increment standard opportunities (simulating one cycle completion)
	s.NoError(s.Setup.Oraclekeeper.IncrementStandardOpportunities(ctx))

	// Set up query opportunities for all cyclelist queries (simulating a full cycle)
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		s.NoError(s.Setup.Oraclekeeper.IncrementQueryOpportunities(ctx, queryId))
	}

	// Track that both reporters reported on all cyclelist queries
	// Both reporters are in the same aggregate for each query
	aggregateTotalPower := reporterPower1 + reporterPower2
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[0], queryId, reporterPower1, aggregateTotalPower, false))
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[1], queryId, reporterPower2, aggregateTotalPower, false))
	}

	// Distribute liveness rewards
	s.NoError(s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx))

	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	// advance height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, err = reporterServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
				SelectorAddress: tc.delegator.String(), ValidatorAddress: sdk.ValAddress(tc.delegator).String(),
			})
			s.NoError(err)
			afterBalance, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, tc.delegator)
			s.NoError(err)
			s.Equal(tc.beforeBalance.Add(tc.afterBalanceIncrease), afterBalance)
		})
	}
}

func (s *IntegrationTestSuite) TestTimeBasedRewardsThreeReporters() {
	qId := utils.QueryIDFromData(ethQueryData)
	ctx := s.Setup.Ctx
	// Accumulated power values for each reporter (simulates multiple reports)
	accPower1 := uint64(2)
	accPower2 := uint64(4)
	accPower3 := uint64(6)
	totalPower := accPower1 + accPower2 + accPower3 // 12
	repAccs, _, _ := s.createValidatorAccs([]uint64{1, 2, 3})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker2")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[2], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker3")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[2], reportertypes.NewSelection(repAccs[2], 1)))
	reporterStake, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[0], qId)
	s.NoError(err)
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[1], qId)
	s.NoError(err)
	reporterStake3, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[2], qId)
	s.NoError(err)

	tipper := s.newKeysWithTokens()
	reward := math.NewInt(100)
	err = s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, tipper, minttypes.TimeBasedRewards, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, reward)))
	s.NoError(err)

	// With liveness-weighted TBR distribution:
	// If all reporters have 100% liveness, reward is proportional to accumulated power
	testCases := []struct {
		name                 string
		beforeBalance        math.Int
		afterBalanceIncrease math.Int
		delegator            sdk.AccAddress
	}{
		{
			name:                 "reporter with accumulated power 2",
			beforeBalance:        reporterStake,
			afterBalanceIncrease: reporterkeeper.CalculateRewardAmount(accPower1, totalPower, reward).TruncateInt(),
			delegator:            repAccs[0],
		},
		{
			name:                 "reporter with accumulated power 4",
			beforeBalance:        reporterStake2,
			afterBalanceIncrease: reporterkeeper.CalculateRewardAmount(accPower2, totalPower, reward).TruncateInt(),
			delegator:            repAccs[1],
		},
		{
			name:                 "reporter with accumulated power 6",
			beforeBalance:        reporterStake3,
			afterBalanceIncrease: reporterkeeper.CalculateRewardAmount(accPower3, totalPower, reward).TruncateInt(),
			delegator:            repAccs[2],
		},
	}

	// Get the cyclelist to set up query opportunities
	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	s.NoError(err)

	// Reset liveness data to clear any setup-time increments (RotateQueries runs during FinalizeBlock)
	s.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))

	// Increment standard opportunities (simulating one cycle completion)
	s.NoError(s.Setup.Oraclekeeper.IncrementStandardOpportunities(ctx))

	// Set up query opportunities for all cyclelist queries
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		s.NoError(s.Setup.Oraclekeeper.IncrementQueryOpportunities(ctx, queryId))
	}

	// Track liveness for all reporters on all cyclelist queries
	// All three reporters are in the same aggregate for each query
	aggregateTotalPower := accPower1 + accPower2 + accPower3
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[0], queryId, accPower1, aggregateTotalPower, false))
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[1], queryId, accPower2, aggregateTotalPower, false))
		s.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, repAccs[2], queryId, accPower3, aggregateTotalPower, false))
	}

	// Distribute liveness rewards
	s.NoError(s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx))

	// advance height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, err = reporterServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
				SelectorAddress: tc.delegator.String(), ValidatorAddress: sdk.ValAddress(tc.delegator).String(),
			})
			s.NoError(err)
			afterBalance, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, tc.delegator)
			s.NoError(err)
			expectedAfterBalance := tc.beforeBalance.Add(tc.afterBalanceIncrease)
			tolerance := expectedAfterBalance.SubRaw(1)
			withinTolerance := expectedAfterBalance.Equal(afterBalance) || tolerance.Equal(afterBalance)
			s.True(withinTolerance)
		})
	}
}

func (s *IntegrationTestSuite) TestTokenBridgeQuery() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx := s.Setup.Ctx
	repAccs, _, _ := s.Setup.CreateValidators(5)
	ok := s.Setup.Oraclekeeper
	m, err := s.Setup.Mintkeeper.Minter.Get(ctx)
	s.NoError(err)
	m.Initialized = true
	s.NoError(s.Setup.Mintkeeper.Minter.Set(ctx, m))

	app := s.Setup.App
	msgServer := keeper.NewMsgServerImpl(ok)
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
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

	_, err = app.BeginBlocker(ctx)
	s.NoError(err)
	_, _ = app.EndBlocker(ctx)

	msgSubmitValue := types.MsgSubmitValue{
		Creator:   reporter1.String(),
		QueryData: querydata,
		Value:     testValue,
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
		Value:     testValue,
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
		Value:     testValue,
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
		Value:     testValue,
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
	// s.Equal(len(agg.Reporters), 4)
	s.Equal(agg.QueryId, crypto.Keccak256(querydata))

	// new report that starts a new cycle
	msgSubmitValue = types.MsgSubmitValue{
		Creator:   reporter5.String(),
		QueryData: querydata,
		Value:     testValue,
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
	s.Equal(agg.AggregateReporter, reporter5.String())

	// Note: WithdrawTip is not tested here because TRBBridge queries don't receive
	// cyclelist TBR rewards (they're not in the cyclelist). This test focuses on
	// verifying the token bridge query submission and aggregation flow.
}

func (s *IntegrationTestSuite) TestTokenBridgeQueryDirectreveal() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	repAccs, _, _ := s.Setup.CreateValidators(5)
	ok := s.Setup.Oraclekeeper
	ctx := s.Setup.Ctx
	app := s.Setup.App
	msgServer := keeper.NewMsgServerImpl(ok)
	for _, rep := range repAccs {
		s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
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
	// s.Equal(len(agg.Reporters), 4)
	s.Equal(agg.QueryId, crypto.Keccak256(querydata))
	s.Equal(agg.AggregateReporter, reporter4.String()) // todo: should it be the last reporter or first reporter

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
	// s.Equal(len(agg.Reporters), 1)
	s.Equal(agg.QueryId, crypto.Keccak256(querydata))
}

// test tipping a query id not in cycle list and observe the reporters' delegators stake increase in staking module
func (s *IntegrationTestSuite) TestTipQueryNotInCycleListSingleDelegator() {
	ctx := s.Setup.Ctx
	require := s.Require()
	ctx = ctx.WithBlockTime(time.Now())
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{1000})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	stakeAmount, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[0], queryId)
	require.NoError(err)
	tipAmount := math.NewInt(100_000)

	tipper := s.newKeysWithTokens()

	valAddr := valAddrs[0]

	// tip. Using msgServer.Tip to handle the transfers and burning of tokens
	msg := types.MsgTip{
		Tipper:    tipper.String(),
		QueryData: queryData,
		Amount:    sdk.NewCoin(s.Setup.Denom, tipAmount),
	}
	_, err = msgServer.Tip(ctx, &msg)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	delBefore, err := s.Setup.Stakingkeeper.Delegation(ctx, repAccs[0].Bytes(), valAddr)
	s.Nil(err)
	s.True(delBefore.GetShares().Equal(math.LegacyNewDecFromInt(stakeAmount)), "delegation shares should be equal to the stake amount")

	reporterPower := uint64(1)
	value := []string{"000001"}
	reports := testutil.GenerateReports(repAccs, value, []uint64{reporterPower}, queryId)
	query, err := s.Setup.Oraclekeeper.CurrentQuery(ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.Nil(s.Setup.Oraclekeeper.Query.Set(ctx, collections.Join(queryId, query.Id), query))
	err = s.Setup.Oraclekeeper.Reports.Set(ctx, collections.Join3(queryId, repAccs[0].Bytes(), query.Id), reports[0])
	s.Nil(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(ctx, query.Id, reports[0]))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(ctx)
	s.Nil(err)

	// check that tip is in escrow
	escrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.Setup.Bankkeeper.GetBalance(ctx, escrowAcct, s.Setup.Denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// create reporterMsgServer
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	// withdraw tip
	_, err = reporterMsgServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
		SelectorAddress: repAccs[0].String(), ValidatorAddress: valAddr.String(),
	})
	require.NoError(err)

	// delegation shares should increase after reporting and escrow balance should go back to 0
	delAfter, err := s.Setup.Stakingkeeper.Delegation(ctx, repAccs[0].Bytes(), valAddr)
	s.Nil(err)
	s.True(delAfter.GetShares().Equal(delBefore.GetShares().Add(math.LegacyNewDec(98000))), "delegation shares plus the tip added") // 100,000 - 2% tip
	escrowBalance = s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, escrowAcct, s.Setup.Denom)
	s.True(escrowBalance.IsZero())
}

func (s *IntegrationTestSuite) TestTipQueryNotInCycleListTwoDelegators() {
	require := s.Require()
	ctx := s.Setup.Ctx
	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{1, 2})
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[0], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker1")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[0], reportertypes.NewSelection(repAccs[0], 1)))

	queryData, _ := utils.QueryBytesFromString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000366696C000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	queryId := utils.QueryIDFromData(queryData)

	reporterStake1, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[0], queryId)
	require.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, repAccs[1], reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker2")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, repAccs[1], reportertypes.NewSelection(repAccs[1], 1)))
	reporterStake2, err := s.Setup.Reporterkeeper.ReporterStake(ctx, repAccs[1], queryId)
	require.NoError(err)

	tipAmount := math.NewInt(100_000)

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
	_, err = msgServer.Tip(ctx, &msg)
	s.Nil(err)

	// check delegation shares before reporting, should be equal to the stake amount
	del1Before, err := s.Setup.Stakingkeeper.Delegation(ctx, delegator1.Bytes(), valAddr1)
	s.Nil(err)
	s.True(del1Before.GetShares().Equal(math.LegacyNewDecFromInt(reporterStake1)), "delegation 1 shares should be equal to the stake amount")

	del2Before, err := s.Setup.Stakingkeeper.Delegation(ctx, delegator2.Bytes(), valAddr2)
	s.Nil(err)
	s.True(del2Before.GetShares().Equal(math.LegacyNewDecFromInt(reporterStake2)), "delegation 2 shares should be equal to the stake amount")

	reporterPower := uint64(1)
	reporterPower2 := uint64(2)
	value := []string{"000001", "000002"}
	reports := testutil.GenerateReports([]sdk.AccAddress{repAccs[0], repAccs[1]}, value, []uint64{reporterPower, reporterPower2}, queryId)
	query, err := s.Setup.Oraclekeeper.CurrentQuery(ctx, queryId)
	s.Nil(err)
	query.HasRevealedReports = true
	s.NoError(s.Setup.Oraclekeeper.Query.Set(ctx, collections.Join(queryId, query.Id), query))
	err = s.Setup.Oraclekeeper.Reports.Set(ctx, collections.Join3(queryId, repAccs[0].Bytes(), query.Id), reports[0])
	s.Nil(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(ctx, query.Id, reports[0]))
	err = s.Setup.Oraclekeeper.Reports.Set(ctx, collections.Join3(queryId, repAccs[1].Bytes(), query.Id), reports[1])
	s.Nil(err)
	s.NoError(s.Setup.Oraclekeeper.AddReport(ctx, query.Id, reports[1]))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3) // bypassing offset that expires time to commit/reveal
	err = s.Setup.Oraclekeeper.SetAggregatedReport(ctx)
	s.Nil(err)

	// check tip escrow account
	escrowAcct := s.Setup.Accountkeeper.GetModuleAddress(reportertypes.TipsEscrowPool)
	require.NotNil(escrowAcct)
	escrowBalance := s.Setup.Bankkeeper.GetBalance(ctx, escrowAcct, s.Setup.Denom)
	require.NotNil(escrowBalance)
	twoPercent := sdk.NewCoin(s.Setup.Denom, tipAmount.Mul(math.NewInt(2)).Quo(math.NewInt(100)))
	require.Equal(tipAmount.Sub(twoPercent.Amount), escrowBalance.Amount)

	// withdraw self delegation from tip escrow
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	_, err = reporterMsgServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
		SelectorAddress: delegator1.String(), ValidatorAddress: valAddr1.String(),
	})
	require.NoError(err)

	// delegation shares should increase after reporting and withdrawing
	del1After, err := s.Setup.Stakingkeeper.Delegation(ctx, delegator1.Bytes(), valAddr1)
	s.Nil(err)
	fmt.Println(del1After.GetShares().String())
	fmt.Println(del1Before.GetShares().String())
	s.True(del1After.GetShares().Equal(del1Before.GetShares().Add(math.LegacyNewDec(32666))), "delegation 1 (self delegation) shares should be half the tip plus 50 percent commission")
	// withdraw del2 delegation from tip escrow
	_, err = reporterMsgServer.WithdrawTip(ctx, &reportertypes.MsgWithdrawTip{
		SelectorAddress: delegator2.String(), ValidatorAddress: valAddr2.String(),
	})
	require.NoError(err)

	del2After, err := s.Setup.Stakingkeeper.Delegation(ctx, delegator2.Bytes(), valAddr2)
	s.Nil(err)
	fmt.Println(del2After.GetShares().String())
	fmt.Println(del2Before.GetShares().String())
	s.True(del2After.GetShares().Equal(del2Before.GetShares().Add(math.LegacyNewDec(65333))), "delegation 2 shares should be half the tip minus 50 percent reporter commission")
}

func (s *IntegrationTestSuite) TestClaimingBridgeDeposit() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(10).WithBlockTime(time.Now())
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(ctx.ConsensusParams())
	reporterMsgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	require.NotNil(reporterMsgServer)
	oracleMsgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	require.NotNil(s.Setup.Bridgekeeper)

	//---------------------------------------------------------------------------
	// Height 10 - create bonded validators and reporters
	//---------------------------------------------------------------------------
	_, err := s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// create 5 validators
	valAccAddrs, valAccountValAddrs, _ := s.Setup.CreateValidators(5)
	for _, val := range valAccountValAddrs {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("addr"))
		s.NoError(err)
	}
	// all 5 validators get free floating tokens and become reporters
	initCoins := sdk.NewCoin(s.Setup.Denom, math.NewInt(500*1e6))
	for i, rep := range valAccAddrs {
		s.NoError(s.Setup.Bankkeeper.MintCoins(s.Setup.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, authtypes.Minter, rep, sdk.NewCoins(initCoins)))

		moniker := "rep" + strconv.Itoa(i)
		msgCreateReporter := reportertypes.MsgCreateReporter{
			ReporterAddress:   rep.String(),
			CommissionRate:    math.LegacyNewDec(1),
			MinTokensRequired: math.NewInt(1 * 1e6),
			Moniker:           moniker,
		}
		_, err := reporterMsgServer.CreateReporter(ctx, &msgCreateReporter)
		require.NoError(err)
	}

	// setup bridge validator checkpoints
	startTime := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	valTimestamp := startTime
	fmt.Println("setting val timestamp: ", valTimestamp)
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("hash"),
		Timestamp:      valTimestamp,
		PowerThreshold: uint64(3000 * 1e6),
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, valTimestamp, checkpointParams))

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 11 - somebody tips and everybody report a bridge deposit
	//---------------------------------------------------------------------------
	ctx = ctx.WithBlockHeight(11).WithBlockTime(time.Now())
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// verify reporters exist
	for _, rep := range valAccAddrs {
		exists, err := s.Setup.Reporterkeeper.Reporters.Has(ctx, rep)
		require.NoError(err)
		require.True(exists)
	}

	// tip bridge deposit
	bridgeQueryDataString := "0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000095452424272696467650000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001"
	bridgeQueryData, _ := hex.DecodeString(bridgeQueryDataString)
	msgTip := types.MsgTip{
		Tipper:    valAccAddrs[0].String(),
		QueryData: bridgeQueryData,
		Amount:    sdk.NewCoin("loya", math.NewInt(1*1e6)),
	}
	_, err = oracleMsgServer.Tip(ctx, &msgTip)
	require.NoError(err)

	// everybody reports the bridge deposit
	// value := layerutil.EncodeValue(10000000)
	value := "0000000000000000000000003386518f7ab3eb51591571adbe62cf94540ead29000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000f424000000000000000000000000000000000000000000000000000000000000003e8000000000000000000000000000000000000000000000000000000000000002d74656c6c6f72317038386a7530796875746d6635703275373938787633756d616137756a77376763683972346600000000000000000000000000000000000000"
	for _, rep := range valAccAddrs {
		msgSubmitValue := types.MsgSubmitValue{
			Creator:   rep.String(),
			QueryData: bridgeQueryData,
			Value:     value,
		}
		_, err = oracleMsgServer.SubmitValue(ctx, &msgSubmitValue)
		require.NoError(err)
	}

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2012 - fast forward blocks, bridge deposit should aggregate
	//---------------------------------------------------------------------------
	ctx = ctx.WithBlockHeight(2012)
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2013 - verify aggregate, fast forward time so 12 hrs expires
	//---------------------------------------------------------------------------
	ctx = ctx.WithBlockHeight(2013)
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	// verify aggregate
	queryId := utils.QueryIDFromData(bridgeQueryData)
	queryServer := keeper.NewQuerier(s.Setup.Oraclekeeper)
	req := types.QueryGetCurrentAggregateReportRequest{
		QueryId: hex.EncodeToString(queryId),
	}
	res, err := queryServer.GetCurrentAggregateReport(ctx, &req)
	require.NotNil(res)
	require.NoError(err)

	// make sure bridge deposit hasnt been claimed yet
	claimed, err := s.Setup.Bridgekeeper.DepositIdClaimedMap.Get(ctx, uint64(1))
	require.ErrorContains(err, "collections: not found")
	require.False(claimed.Claimed)

	// fast forward 12 hrs so deposit can be claimed
	ctx = ctx.WithBlockTime(time.Now().Add(13 * time.Hour))

	// try to claim manually -- works fine
	// fmt.Println("agg timestamp: ", res.Timestamp)
	// require.NoError(s.Setup.Bridgekeeper.ClaimDeposit(ctx, uint64(1), res.Timestamp))
	// claimed, err = s.Setup.Bridgekeeper.DepositIdClaimedMap.Get(ctx, uint64(1))
	// require.NoError(err)
	// require.True(claimed.Claimed)

	// try to call AutoClaim manually -- works fine
	// require.NoError(s.Setup.Oraclekeeper.AutoClaimDeposits(ctx))
	// claimed, err = s.Setup.Bridgekeeper.DepositIdClaimedMap.Get(ctx, uint64(1))
	// require.NoError(err)
	// require.True(claimed.Claimed)

	// deposit should get autoclaimed in endblocker -- app.Endblocker panicking from nil k.bridgekeeper (something up with setup)
	err = oracle.EndBlocker(ctx, s.Setup.Oraclekeeper)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2014 - verify deposit is claimed
	//---------------------------------------------------------------------------
	ctx = ctx.WithBlockHeight(2014)
	_, err = s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	claimed, err = s.Setup.Bridgekeeper.DepositIdClaimedMap.Get(ctx, uint64(1))
	require.NoError(err)
	require.True(claimed.Claimed)

	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)
}

func (s *IntegrationTestSuite) TestRotateQueriesToExpiredTippedQuery() {
	require := s.Require()
	ctx := s.Setup.Ctx
	app := s.Setup.App
	okpr := s.Setup.Oraclekeeper
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err := app.BeginBlocker(ctx)
	require.NoError(err)
	msgServer := keeper.NewMsgServerImpl(okpr)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	for _, val := range valAddrs {
		err = s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	_, err = app.EndBlocker(ctx)
	require.NoError(err)
	addr := s.newKeysWithTokens()
	// test for rotating queries going through the cycle list and updating the current query 1,2,3
	// get cycle list
	cycleList, err := okpr.GetCyclelist(ctx)
	s.NoError(err)
	s.Len(cycleList, 3)
	queryId0 := utils.QueryIDFromData(cycleList[0])
	queryId1 := utils.QueryIDFromData(cycleList[1])
	// should be on the second query since the first one is expired from chain running during setup
	query1, err := okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query1, cycleList[1]))
	require.NoError(err)
	require.True(bytes.Equal(query1, cycleList[1]))
	query, err := okpr.CurrentQuery(ctx, queryId1)
	s.NoError(err)
	s.Equal(uint64(3), query.Expiration)
	// queryId1 may or may not have a tip or be in cycle list, but that's not what we're testing

	// Get the initial meta ID of queryId0 before tipping
	// queryId0 is not the current query (queryId1 is current)
	initialQuery0, err := okpr.CurrentQuery(ctx, queryId0)
	var initialMetaId0 uint64
	if err == nil {
		initialMetaId0 = initialQuery0.Id
	} else {
		// If query doesn't exist yet, it will be initialized with a new ID when rotated to
		// We'll track this differently
		initialMetaId0 = 0
	}

	// Tip queryId0 when it is NOT the current cycle list query (queryId1 is current)
	// This simulates tipping a query that is in the cycle list but not currently active
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: cycleList[0], // tip queryId0 which is not current
		Amount:    sdk.NewCoin("loya", math.NewInt(1*1e6)),
	}
	_, err = msgServer.Tip(ctx, &msg)
	require.NoError(err)

	// Verify the query was tipped but is still not current
	query, err = okpr.CurrentQuery(ctx, queryId0)
	require.NoError(err)
	tippedMetaId0 := query.Id
	s.True(query.Amount.GT(math.ZeroInt()), "Query should have a tip")
	s.False(query.CycleList, "Query should not be in cycle list yet")

	// Verify queryId0 expiration after tipping (should be blockHeight + 2 = 3)
	query, err = okpr.CurrentQuery(ctx, queryId0)
	require.NoError(err)
	queryExpiration := query.Expiration
	s.Equal(uint64(3), queryExpiration, "Query should expire at block 3")

	// Move forward in blocks without anyone reporting for queryId0
	// This will cause queryId0 to expire without reports
	// First, let queryId1 expire and rotate to queryId2
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1) // block 2
	_, err = app.BeginBlocker(ctx)
	require.NoError(err)
	_, err = app.EndBlocker(ctx)
	require.NoError(err)
	s.Equal(int64(2), ctx.BlockHeight())

	// queryId1 hasn't expired yet, should still be on queryId1
	query1, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query1, cycleList[1]))

	// Move forward so queryId2 expires and rotates to queryId0
	// queryId2 becomes current at block 3 and expires at block 5 (block 3 + 2 window)
	// queryId0 was tipped but no one reported for it, so it should have expired by now
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1) // block 3
	_, err = app.BeginBlocker(ctx)
	require.NoError(err)
	_, err = app.EndBlocker(ctx)
	require.NoError(err)
	s.Equal(int64(3), ctx.BlockHeight())

	// queryId2 is now current, wait for it to expire
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1) // block 4
	_, err = app.BeginBlocker(ctx)
	require.NoError(err)
	_, err = app.EndBlocker(ctx)
	require.NoError(err)
	s.Equal(int64(4), ctx.BlockHeight())

	// queryId2 expires at block 5, so rotate to queryId0
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1) // block 5
	_, err = app.BeginBlocker(ctx)
	require.NoError(err)
	_, err = app.EndBlocker(ctx)
	require.NoError(err)
	s.Equal(int64(5), ctx.BlockHeight())
	s.Greater(int64(5), int64(queryExpiration), "QueryId0 should have expired by block 5")

	// Should now be on queryId0 since queryId2 expired
	query1, err = okpr.GetCurrentQueryInCycleList(ctx)
	s.NoError(err)
	s.True(bytes.Equal(query1, cycleList[0]))

	// Verify that queryId0 is now the current query and its meta ID has increased
	// because it was tipped but not reported, and is now rotated to as part of the cycle list
	query, err = okpr.CurrentQuery(ctx, queryId0)
	require.NoError(err)
	s.True(query.CycleList, "Query should now be in cycle list")
	s.True(query.Amount.GT(math.ZeroInt()), "Query should still have the tip")

	// The meta ID should have increased when the expired tipped query was rotated to
	if initialMetaId0 > 0 {
		s.Greater(query.Id, initialMetaId0, "Meta ID should have increased when expired tipped query was rotated to")
	}
	// Also verify it's greater than the ID it had when tipped
	s.Greater(query.Id, tippedMetaId0, "Meta ID should have increased when expired tipped query was rotated to")
}

func (s *IntegrationTestSuite) TestSubmitMultipleReportsDifferentMetaIds() {
	require := s.Require()
	ctx := s.Setup.Ctx
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	ctx = ctx.WithBlockTime(time.Now())
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err := s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200})
	for _, val := range valAddrs {
		require.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real")))
	}
	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	tipper := s.newKeysWithTokens()

	// register reporters
	for _, rep := range repAccs {
		require.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter")))
		require.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		require.NoError(err)
	}

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))
	queryDatas := [][]byte{ethQueryData, btcQueryData, trbQueryData}
	metaIds := make([]uint64, 3)

	for i, qd := range queryDatas {
		// tip the query
		_, err := msgServer.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: qd, Amount: tip})
		require.NoError(err)

		queryId := utils.QueryIDFromData(qd)
		query, err := s.Setup.Oraclekeeper.CurrentQuery(ctx, queryId)
		require.NoError(err)
		metaIds[i] = query.Id

		// both reporters submit
		for _, rep := range repAccs {
			_, err = msgServer.SubmitValue(ctx, &types.MsgSubmitValue{
				Creator:   rep.String(),
				QueryData: qd,
				Value:     testValue,
			})
			require.NoError(err)
		}

		// advance blocks to expire and aggregate
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
		_, err = s.Setup.App.BeginBlocker(ctx)
		require.NoError(err)
		_, err = s.Setup.App.EndBlocker(ctx)
		require.NoError(err)
		require.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))

		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// verify all metaIds are different
	require.NotEqual(metaIds[0], metaIds[1], "eth and btc should have different metaIds")
	require.NotEqual(metaIds[1], metaIds[2], "btc and trb should have different metaIds")
	require.NotEqual(metaIds[0], metaIds[2], "eth and trb should have different metaIds")

	// verify reports exist for each query
	for i, qd := range queryDatas {
		queryId := utils.QueryIDFromData(qd)
		for _, rep := range repAccs {
			report, err := s.Setup.Oraclekeeper.Reports.Get(ctx, collections.Join3(queryId, rep.Bytes(), metaIds[i]))
			require.NoError(err)
			require.Equal(rep.String(), report.Reporter)
		}
	}

	fmt.Printf("MetaIds - ETH/USD: %d, BTC/USD: %d, TRB/USD: %d\n", metaIds[0], metaIds[1], metaIds[2])
}

func (s *IntegrationTestSuite) TestRemoveOldReports() {
	require := s.Require()
	ctx := s.Setup.Ctx
	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})

	now := time.Now()
	ctx = ctx.WithBlockTime(now)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err := s.Setup.App.BeginBlocker(ctx)
	require.NoError(err)

	msgServer := keeper.NewMsgServerImpl(s.Setup.Oraclekeeper)
	repAccs, valAddrs, _ := s.createValidatorAccs([]uint64{100, 200})
	for _, val := range valAddrs {
		require.NoError(s.Setup.Bridgekeeper.SetEVMAddressByOperator(ctx, val.String(), []byte("not real")))
	}
	_, err = s.Setup.App.EndBlocker(ctx)
	require.NoError(err)

	tipper := s.newKeysWithTokens()

	for _, rep := range repAccs {
		require.NoError(s.Setup.Reporterkeeper.Reporters.Set(ctx, rep, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter")))
		require.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, rep, reportertypes.NewSelection(rep, 1)))
		_, err := s.Setup.Reporterkeeper.ReporterStake(ctx, rep, []byte{})
		require.NoError(err)
	}

	tip := sdk.NewCoin(s.Setup.Denom, math.NewInt(100_000))

	// helper to count all reports in the store
	countReports := func(c sdk.Context) int {
		count := 0
		_ = s.Setup.Oraclekeeper.Reports.Walk(c, nil, func(_ collections.Triple[[]byte, []byte, uint64], _ types.MicroReport) (bool, error) {
			count++
			return false, nil
		})
		return count
	}

	// --- Round 1: submit reports 45 days ago (should be pruned) ---
	oldTime := now.Add(-45 * 24 * time.Hour)
	ctx = ctx.WithBlockTime(oldTime)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	_, err = msgServer.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: ethQueryData, Amount: tip})
	require.NoError(err)
	for _, rep := range repAccs {
		_, err = msgServer.SubmitValue(ctx, &types.MsgSubmitValue{Creator: rep.String(), QueryData: ethQueryData, Value: testValue})
		require.NoError(err)
	}
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	require.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))

	// --- submit reports 31 days ago (should be pruned) ---
	oldTime2 := now.Add(-31 * 24 * time.Hour)
	ctx = ctx.WithBlockTime(oldTime2)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	_, err = msgServer.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: btcQueryData, Amount: tip})
	require.NoError(err)
	for _, rep := range repAccs {
		_, err = msgServer.SubmitValue(ctx, &types.MsgSubmitValue{Creator: rep.String(), QueryData: btcQueryData, Value: testValue})
		require.NoError(err)
	}
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	require.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))

	// --- submit reports 5 days ago (should NOT be pruned) ---
	recentTime := now.Add(-5 * 24 * time.Hour)
	ctx = ctx.WithBlockTime(recentTime)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	_, err = msgServer.Tip(ctx, &types.MsgTip{Tipper: tipper.String(), QueryData: trbQueryData, Amount: tip})
	require.NoError(err)
	for _, rep := range repAccs {
		_, err = msgServer.SubmitValue(ctx, &types.MsgSubmitValue{Creator: rep.String(), QueryData: trbQueryData, Value: testValue})
		require.NoError(err)
	}
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	require.NoError(s.Setup.Oraclekeeper.SetAggregatedReport(ctx))

	// We should have 6 reports total: 2 reporters × 3 queries
	totalBefore := countReports(ctx)
	require.Equal(6, totalBefore, "should have 6 reports before pruning")

	// Prune with current time (now)
	// All 4 old reports deleted
	ctx = ctx.WithBlockTime(now)
	require.NoError(s.Setup.Oraclekeeper.RemoveOldReports(ctx))
	require.Equal(2, countReports(ctx), "should have 2 reports after pruning old ones")

	// Prune again, nothing left to prune
	require.NoError(s.Setup.Oraclekeeper.RemoveOldReports(ctx))
	require.Equal(2, countReports(ctx), "recent reports should not be pruned")

	// --- Verify the remaining reports are the TRB/USD ones ---
	trbQueryId := utils.QueryIDFromData(trbQueryData)
	for _, rep := range repAccs {
		has, err := s.Setup.Oraclekeeper.Reports.Has(ctx, collections.Join3(trbQueryId, rep.Bytes(), uint64(2)))
		require.NoError(err)
		require.True(has, "TRB/USD report should still exist")
	}

	// --- Verify old reports are gone ---
	ethQueryId := utils.QueryIDFromData(ethQueryData)
	btcQueryId := utils.QueryIDFromData(btcQueryData)
	for _, rep := range repAccs {
		has, err := s.Setup.Oraclekeeper.Reports.Has(ctx, collections.Join3(ethQueryId, rep.Bytes(), uint64(0)))
		require.NoError(err)
		require.False(has, "ETH/USD report should be pruned")

		has, err = s.Setup.Oraclekeeper.Reports.Has(ctx, collections.Join3(btcQueryId, rep.Bytes(), uint64(1)))
		require.NoError(err)
		require.False(has, "BTC/USD report should be pruned")
	}

	// --- Verify indexes are also cleaned up ---
	indexCount := 0
	iter, err := s.Setup.Oraclekeeper.Reports.Indexes.IdQueryId.Iterate(ctx, nil)
	require.NoError(err)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		indexCount++
	}
	require.Equal(2, indexCount, "IdQueryId index should only have 2 entries after pruning")
}
