package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestVote() {
	// Add dispute
	addr := s.TestMsgProposeDisputeFromAccount()

	// mock dependency modules
	s.bankKeeper.On("GetBalance", s.ctx, addr, layer.BondDenom).Return(sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.oracleKeeper.On("GetUserTips", s.ctx, addr).Return(oracletypes.UserTipTotal{Address: addr.String(), Total: math.NewInt(1)}, nil)
	s.bankKeeper.On("GetSupply", s.ctx, layer.BondDenom).Return(sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.oracleKeeper.On("GetTotalTips", s.ctx).Return(math.NewInt(1), nil)
	s.reporterKeeper.On("TotalReporterPower", s.ctx).Return(math.NewInt(1), nil)

	voteMsg := types.MsgVote{
		Voter: addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}
	// vote should have started
	_, err := s.msgServer.Vote(s.ctx, &voteMsg)
	s.NoError(err)

	_, err = s.msgServer.Vote(s.ctx, &voteMsg)
	s.Error(err)

	voterVote, err := s.disputeKeeper.GetVoterVote(s.ctx, addr.String(), 1)
	s.NoError(err)

	s.Equal(voterVote.Voter, addr.String())
	s.Equal(voterVote.Id, uint64(1))
	s.Equal(voterVote.Vote, types.VoteEnum_VOTE_SUPPORT)

	// start voting, this method is check on beginblock
	vote, err := s.disputeKeeper.GetVote(s.ctx, 1)
	s.NoError(err)
	s.NotNil(vote)
	s.Equal(vote.Voters, []string{addr.String()})
	s.Equal(vote.VoteResult, types.VoteResult_NO_TALLY)
	s.Equal(vote.Id, uint64(1))
}
