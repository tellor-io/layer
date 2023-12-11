package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestVote() {
	require := s.Require()
	// Add dispute
	s.TestMsgProposeDisputeFromAccount()
	s.bankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything).Return(sdk.NewCoin("trb", math.NewInt(1)))
	s.oracleKeeper.On("GetUserTips", mock.Anything, mock.Anything).Return(oracletypes.UserTipTotal{Address: "", Total: sdk.NewCoin("trb", math.NewInt(1))})
	s.stakingKeeper.On("GetLastTotalPower", mock.Anything).Return(sdk.NewInt(1))
	s.bankKeeper.On("GetSupply", mock.Anything, mock.Anything).Return(sdk.NewCoin("trb", math.NewInt(1)))
	s.oracleKeeper.On("GetTotalTips", mock.Anything, mock.Anything).Return(sdk.NewCoin("trb", math.NewInt(1)))
	voteMsg := types.MsgVote{
		Voter: Addr.String(),
		Id:    1,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	}
	// vote should have started
	_, err := s.msgServer.Vote(s.goCtx, &voteMsg)
	require.NoError(err)

	_, err = s.msgServer.Vote(s.goCtx, &voteMsg)
	require.Error(err)

	voterVote := s.disputeKeeper.GetVoterVote(s.ctx, Addr.String(), 1)
	require.Equal(voterVote.Voter, Addr.String())
	require.Equal(voterVote.Id, uint64(1))
	require.Equal(voterVote.Vote, types.VoteEnum_VOTE_SUPPORT)

	// start voting, this method is check on beginblock
	vote := s.disputeKeeper.GetVote(s.ctx, 1)
	require.NotNil(vote)
	require.Equal(vote.Voters, []string{Addr.String()})
	require.Equal(vote.VoteResult, types.VoteResult_NO_TALLY)
	require.Equal(vote.Id, uint64(1))
}
