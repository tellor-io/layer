package integration_test

import (
	"encoding/hex"
	"time"

	"testing"

	"github.com/cosmos/cosmos-sdk/x/gov"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (s *IntegrationTestSuite) oracleKeeper() (queryClient types.QueryClient, msgServer types.MsgServer) {
	types.RegisterQueryServer(s.queryHelper, s.oraclekeeper)
	types.RegisterInterfaces(s.interfaceRegistry)
	queryClient = types.NewQueryClient(s.queryHelper)
	msgServer = keeper.NewMsgServerImpl(s.oraclekeeper)
	return
}

func (s *IntegrationTestSuite) TestTipping() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	store := s.oraclekeeper.TipStore(s.ctx)
	tips, _ := s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	s.Equal(tips.QueryData, ethQueryData[2:])
	s.Equal(tip.Sub(twoPercent), tips.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	userTips := s.oraclekeeper.GetUserQueryTips(s.ctx, addr.String(), ethQueryData)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)

	// tip same query again
	_, err = msgServer.Tip(s.ctx, &msg)
	s.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, ethQueryData)
	s.Equal(tips.QueryData, ethQueryData[2:])
	// tips should be 2x
	s.Equal(tip.Sub(twoPercent).Amount.Mul(sdk.NewInt(2)), tips.Amount.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	// total tips overall
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)

	// tip different query
	_, err = msgServer.Tip(s.ctx, &types.MsgTip{QueryData: btcQueryData, Tipper: addr.String(), Amount: tip})
	s.NoError(err)
	tips, _ = s.oraclekeeper.GetQueryTips(s.ctx, store, btcQueryData)
	s.Equal(tips.QueryData, btcQueryData[2:])
	s.Equal(tip.Sub(twoPercent), tips.Amount)
	s.Equal(tips.TotalTips, tips.Amount)
	userTips = s.oraclekeeper.GetUserQueryTips(s.ctx, addr.String(), btcQueryData)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount)
	userTips = s.oraclekeeper.GetUserTips(s.ctx, addr)
	s.Equal(userTips.Address, addr.String())
	s.Equal(userTips.Total, tips.Amount.Add(tips.Amount).Add(tips.Amount))
}

func (s *IntegrationTestSuite) TestGetCurrentTip() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
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
	s.Equal(resp.Tips, &types.Tips{QueryData: ethQueryData[2:], Amount: tip.Sub(twoPercent), TotalTips: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestGetUserTipTotal() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(1000))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
	msg := types.MsgTip{
		Tipper:    addr.String(),
		QueryData: ethQueryData,
		Amount:    tip,
	}
	_, err := msgServer.Tip(s.ctx, &msg)
	s.NoError(err)

	// Get current tip
	resp, err := s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String(), QueryData: ethQueryData})
	s.NoError(err)
	s.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
	// Check total tips without a given query data
	resp, err = s.oraclekeeper.GetUserTipTotal(s.ctx, &types.QueryGetUserTipTotalRequest{Tipper: addr.String()})
	s.NoError(err)
	s.Equal(resp.TotalTips, &types.UserTipTotal{Address: addr.String(), Total: tip.Sub(twoPercent)})
}

func (s *IntegrationTestSuite) TestSmallTip() {
	_, msgServer := s.oracleKeeper()
	addr := s.newKeysWithTokens()
	tip := sdk.NewCoin(s.denom, sdk.NewInt(10))
	twoPercent := sdk.NewCoin(s.denom, tip.Amount.Mul(sdk.NewInt(2)).Quo(sdk.NewInt(100)))
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
	_, msgServer := s.oracleKeeper()
	accs, _, privKeys := s.createValidatorAccs([]int64{100, 200, 300, 400, 500})
	reporters := []struct {
		name          string
		reporterIndex int
		value         string
	}{
		{
			name:          "reporter 1",
			reporterIndex: 0,
			value:         encodeValue(162926),
		},
		{
			name:          "reporter 2",
			reporterIndex: 1,
			value:         encodeValue(362926),
		},
		{
			name:          "reporter 3",
			reporterIndex: 2,
			value:         encodeValue(262926),
		},
		{
			name:          "reporter 4",
			reporterIndex: 3,
			value:         encodeValue(562926),
		},
		{
			name:          "reporter 5",
			reporterIndex: 4,
			value:         encodeValue(462926),
		},
	}
	for _, r := range reporters {
		s.T().Run(r.name, func(t *testing.T) {
			valueDecoded, err := hex.DecodeString(r.value) // convert hex value to bytes
			s.Nil(err)
			signature, err := privKeys[r.reporterIndex].Sign(valueDecoded) // sign value
			s.Nil(err)
			commit, reveal := report(accs[r.reporterIndex].String(), hex.EncodeToString(signature), r.value, ethQueryData)
			_, err = msgServer.CommitReport(s.ctx, &commit)
			s.Nil(err)
			_, err = msgServer.SubmitValue(s.ctx.WithBlockHeight(s.ctx.BlockHeight()+1), &reveal)
			s.Nil(err)
		})
	}
	s.ctx = s.ctx.WithBlockHeight(s.ctx.BlockHeight() + 1)
	s.app.EndBlocker(s.ctx, abci.RequestEndBlock{Height: s.ctx.BlockHeight()})
	// check median
	qId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	res, err := s.oraclekeeper.GetAggregatedReport(s.ctx, &types.QueryGetCurrentAggregatedReportRequest{QueryId: qId})
	s.Nil(err)
	expectedMedianReporterIndex := 4
	expectedMedianReporter := accs[expectedMedianReporterIndex].String()
	s.Equal(expectedMedianReporter, res.Report.AggregateReporter)
	s.Equal(reporters[expectedMedianReporterIndex].value, res.Report.AggregateValue)
}

func report(creator, signature, value, qdata string) (oracletypes.MsgCommitReport, oracletypes.MsgSubmitValue) {
	commit := oracletypes.MsgCommitReport{
		Creator:   creator,
		QueryData: ethQueryData,
		Signature: signature,
	}
	reveal := oracletypes.MsgSubmitValue{
		Creator:   creator,
		QueryData: ethQueryData,
		Value:     value,
	}
	return commit, reveal
}

func (s *IntegrationTestSuite) TestGetCylceListQueries() {
	s.oracleKeeper()
	accs, _, _ := s.createValidatorAccs([]int64{100, 200, 300, 400, 500})
	// Get supported queries
	resp := s.oraclekeeper.GetCyclList(s.ctx)
	s.Equal(resp.QueryData, []string{ethQueryData, btcQueryData, trbQueryData})
	fakeQueryData := "0x000001"
	p := oracletypes.CycleListChangeProposal{
		Title:       "Test",
		Description: "test description",
		NewList:     []string{fakeQueryData},
	}
	msgContent, err := v1.NewLegacyContent(&p, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	s.NoError(err)
	proposal1, err := s.govKeeper.SubmitProposal(s.ctx, []sdk.Msg{msgContent}, "", "test", "description", accs[0])
	s.NoError(err)

	votingStarted, err := s.govKeeper.AddDeposit(s.ctx, proposal1.Id, accs[0], s.govKeeper.GetParams(s.ctx).MinDeposit)
	s.NoError(err)
	s.True(votingStarted)
	proposal1, ok := s.govKeeper.GetProposal(s.ctx, proposal1.Id)
	s.True(ok)
	s.True(proposal1.Status == v1.StatusVotingPeriod)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[0], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[1], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	err = s.govKeeper.AddVote(s.ctx, proposal1.Id, accs[2], v1.NewNonSplitVoteOption(v1.OptionYes), "")
	s.NoError(err)
	proposal1, ok = s.govKeeper.GetProposal(s.ctx, proposal1.Id)
	s.True(ok)
	s.ctx = s.ctx.WithBlockTime(s.ctx.BlockTime().Add(time.Hour * 24 * 2))
	gov.EndBlocker(s.ctx, s.govKeeper)
	proposal1, _ = s.govKeeper.GetProposal(s.ctx, proposal1.Id)
	s.True(proposal1.Status == v1.StatusPassed)
	resp = s.oraclekeeper.GetCyclList(s.ctx)
	s.Equal(resp.QueryData, []string{fakeQueryData})
}
