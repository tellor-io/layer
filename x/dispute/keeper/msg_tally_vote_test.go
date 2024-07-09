package keeper_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgTallyVote() {
	require := s.Require()
	msgServer := s.msgServer
	ctx := s.ctx
	k := s.disputeKeeper
	bk := s.bankKeeper

	res, err := msgServer.TallyVote(s.ctx, &types.MsgTallyVote{
		CallerAddress: "caller_address",
		DisputeId:     uint64(1),
	})
	require.Error(err)
	require.Nil(res)

	id1 := uint64(1)
	userAddr := sample.AccAddressBytes()
	tokenHolderAddr := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	teamAddr, err := k.GetTeamAddress(ctx)
	require.NoError(err)

	require.NoError(k.Votes.Set(ctx, id1, types.Vote{
		Id:         id1,
		VoteResult: types.VoteResult_NO_TALLY,
		Executed:   false,
	}))

	require.NoError(k.Disputes.Set(ctx, id1, types.Dispute{
		DisputeId: id1,
		HashId:    []byte("hashId"),
	}))

	require.NoError(k.BlockInfo.Set(ctx, []byte("hashId"), types.BlockInfo{
		TotalReporterPower: math.NewInt(250 * 1e6),
		TotalUserTips:      math.NewInt(250 * 1e6),
	}))
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil)
	bk.On("GetSupply", ctx, layertypes.BondDenom).Return(sdk.Coin{Denom: layertypes.BondDenom, Amount: math.NewInt(250 * 1e6)}, nil)
	require.ErrorContains(k.TallyVote(ctx, id1), "vote period not ended and quorum not reached")

	// set team vote (25% support)
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, teamAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))
	_, err = k.SetTeamVote(ctx, id1, teamAddr)
	require.NoError(err)
	bk.On("GetBalance", ctx, teamAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, userAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, reporterAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	bk.On("GetBalance", ctx, tokenHolderAddr, layertypes.BondDenom).Return(sdk.NewCoin(layertypes.BondDenom, math.NewInt(250*1e6)), nil).Once()
	require.Error(k.TallyVote(ctx, id1))

	// set user vote (25% support)
	require.NoError(k.UsersGroup.Set(ctx, collections.Join(id1, userAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, userAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))

	// set reporter vote (25% support)
	require.NoError(k.ReportersGroup.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), math.NewInt(250*1e6)))
	require.NoError(k.Voter.Set(ctx, collections.Join(id1, reporterAddr.Bytes()), types.Voter{Vote: types.VoteEnum_VOTE_SUPPORT, VoterPower: math.NewInt(250 * 1e6)}))

	res, err = msgServer.TallyVote(s.ctx, &types.MsgTallyVote{
		CallerAddress: "caller_address",
		DisputeId:     uint64(1),
	})
	require.NoError(err)
	require.NotNil(res)
}
