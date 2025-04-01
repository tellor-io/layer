package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestVote() {
	// k := s.disputeKeeper
	// Create dispute
	addr, dispute := s.TestMsgProposeDisputeFromAccount()

	s.oracleKeeper.On("GetTipsAtBlockForTipper", s.ctx, uint64(s.ctx.BlockHeight()), addr).Return(math.NewInt(10), nil)
	s.reporterKeeper.On("Delegation", s.ctx, addr).Return(reportertypes.Selection{
		LockedUntilTime:  time.Now().Add(-1 * time.Hour),
		Reporter:         addr.Bytes(),
		DelegationsCount: 1,
	}, nil)
	s.reporterKeeper.On("GetReporterTokensAtBlock", s.ctx, addr.Bytes(), uint64(s.ctx.BlockHeight())).Return(math.NewInt(10), nil)

	// need to manually call setblock info - happens in endblock normally
	err := s.disputeKeeper.SetBlockInfo(s.ctx, dispute.HashId)
	s.NoError(err)
	voteMsg := types.MsgVote{
		Voter: addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}
	// vote should have started
	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.Error(err)

	voterVote, err := s.disputeKeeper.Voter.Get(s.ctx, collections.Join(uint64(1), addr.Bytes()))
	s.NoError(err)

	s.Equal(voterVote.Vote, types.VoteEnum_VOTE_SUPPORT)

	// start voting, this method is check on beginblock
	vote, err := s.disputeKeeper.Votes.Get(s.ctx, 1)
	s.NoError(err)
	s.NotNil(vote)
	iter, err := s.disputeKeeper.Voter.Indexes.VotersById.MatchExact(s.ctx, uint64(1))
	s.NoError(err)
	keys, err := iter.PrimaryKeys()
	s.NoError(err)
	s.Equal(keys[0].K2(), addr.Bytes())

	// vote from team
	teamAddr, err := s.disputeKeeper.GetTeamAddress(s.ctx)
	s.NoError(err)
	_, err = s.disputeKeeper.SetTeamVote(s.ctx, uint64(1), teamAddr, types.VoteEnum_VOTE_SUPPORT)
	s.NoError(err)

	// check on voting tally
	_, err = s.disputeKeeper.VoteCountsByGroup.Get(s.ctx, uint64(1))
	s.NoError(err)
	// vote calls tally, enough ppl have voted to reach quorum
	s.Equal(vote.VoteResult, types.VoteResult_SUPPORT)
	s.Equal(vote.Id, uint64(1))
}

func BenchmarkMsgVote(b *testing.B) {
	// setup keepers
	disputeKeeper, oracleKeeper, reporterKeeper, _, bankKeeper, ctx := keepertest.DisputeKeeper(b)
	msgServer := keeper.NewMsgServerImpl(disputeKeeper)
	ctx = ctx.WithBlockTime(time.Now())

	// Create initial dispute using ProposeDispute
	addr := sample.AccAddressBytes()
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  addr.String(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Now().Add(-1 * 24 * time.Hour),
		Power:     1,
		MetaId:    1,
	}

	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(10000))
	disputeMsg := types.MsgProposeDispute{
		Creator:          addr.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		DisputeCategory:  types.Warning,
		Fee:              fee,
		PayFromBond:      false,
	}

	// Setup mock expectations for dispute creation
	reporterKeeper.On("EscrowReporterStake", mock.Anything, addr, uint64(1), uint64(0), math.NewInt(10_000), qId, mock.Anything).Return(nil)
	reporterKeeper.On("TotalReporterPower", mock.Anything).Return(math.NewInt(1), nil)
	oracleKeeper.On("GetTotalTips", mock.Anything).Return(math.NewInt(1), nil)
	reporterKeeper.On("JailReporter", mock.Anything, addr, uint64(0)).Return(nil)
	bankKeeper.On("HasBalance", mock.Anything, addr, fee).Return(true)
	bankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, addr, mock.Anything, sdk.NewCoins(fee)).Return(nil)
	oracleKeeper.On("FlagAggregateReport", mock.Anything, report).Return(nil)
	oracleKeeper.On("ValidateMicroReportExists", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&report, true, nil)

	// Setup mock expectations for voting
	oracleKeeper.On("GetTipsAtBlockForTipper", mock.Anything, mock.Anything, addr).Return(math.NewInt(10), nil)
	reporterKeeper.On("Delegation", mock.Anything, addr).Return(reportertypes.Selection{
		LockedUntilTime:  time.Now().Add(-1 * time.Hour),
		Reporter:         addr.Bytes(),
		DelegationsCount: 1,
	}, nil)
	reporterKeeper.On("GetReporterTokensAtBlock", mock.Anything, addr.Bytes(), mock.Anything).Return(math.NewInt(10), nil)

	voteMsg := types.MsgVote{
		Voter: addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		iterationCtx := ctx.WithBlockTime(time.Now())

		// Create dispute
		_, err := msgServer.ProposeDispute(iterationCtx, &disputeMsg)
		if err != nil {
			b.Fatal(err)
		}

		// get dispute hash id
		dispute, err := disputeKeeper.Disputes.Get(iterationCtx, uint64(1))
		if err != nil {
			b.Fatal(err)
		}

		// Set block info (normally happens in endblock)
		err = disputeKeeper.SetBlockInfo(iterationCtx, dispute.HashId)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		// Submit vote
		_, err = msgServer.Vote(iterationCtx, &voteMsg)
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		// Cleanup for next iteration
		err = disputeKeeper.Disputes.Remove(iterationCtx, uint64(1))
		if err != nil {
			b.Fatal(err)
		}
		err = disputeKeeper.Votes.Remove(iterationCtx, uint64(1))
		if err != nil {
			b.Fatal(err)
		}
		err = disputeKeeper.Voter.Remove(iterationCtx, collections.Join(uint64(1), addr.Bytes()))
		if err != nil {
			b.Fatal(err)
		}
		err = disputeKeeper.VoteCountsByGroup.Remove(iterationCtx, uint64(1))
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}
