package integration_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
)

func (s *IntegrationTestSuite) disputeKeeper() (queryClient types.QueryClient, msgServer types.MsgServer) {
	types.RegisterQueryServer(s.queryHelper, s.disputekeeper)
	queryClient = types.NewQueryClient(s.queryHelper)
	types.RegisterInterfaces(s.interfaceRegistry)

	msgServer = keeper.NewMsgServerImpl(s.disputekeeper)
	return
}

func (s *IntegrationTestSuite) TestVotingOnDispute() {
	_, msgServer := s.disputeKeeper()
	require := s.Require()
	ctx := s.ctx
	k := s.disputekeeper
	Addr, denom := s.newKeysWithTokens()

	report, valAddr := s.microReport()
	// Propose dispute pay half of the fee from account
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         Addr.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(denom, sdk.NewInt(5000)),
		DisputeCategory: types.Warning,
	})
	require.Equal(uint64(1), k.GetDisputeCount(ctx))
	require.Equal(1, len(k.GetOpenDisputeIds(ctx).Ids))
	require.NoError(err)
	// check validator wasn't slashed/jailed
	val, found := s.stakingKeeper.GetValidator(ctx, valAddr)
	bondedTokensBefore := val.GetBondedTokens()
	require.True(found)
	require.False(val.IsJailed())
	require.Equal(bondedTokensBefore, sdk.NewInt(1000000))
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = msgServer.AddFeeToDispute(ctx, &types.MsgAddFeeToDispute{
		Creator:   Addr.String(),
		DisputeId: 0,
		Amount:    sdk.NewCoin(denom, sdk.NewInt(5000)),
	})
	require.NoError(err)
	// check validator was slashed/jailed
	val, found = s.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)
	require.True(val.IsJailed())
	// check validator was slashed 1% of tokens
	require.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))))
	dispute := k.GetDisputeById(ctx, 0)
	require.Equal(types.Prevote, dispute.DisputeStatus)
	// these are called during begin block
	ids := k.CheckPrevoteDisputesForExpiration(ctx)
	k.StartVoting(ctx, ids)
	dispute = k.GetDisputeById(ctx, 0)
	require.Equal(types.Voting, dispute.DisputeStatus)
	// vote on dispute
	_, err = msgServer.Vote(ctx, &types.MsgVote{
		Voter: Addr.String(),
		Id:    0,
		Vote:  types.VoteEnum_VOTE_SUPPORT,
	})
	require.NoError(err)
	voterV := k.GetVoterVote(ctx, Addr.String(), 0)
	require.Equal(types.VoteEnum_VOTE_SUPPORT, voterV.Vote)
	v := k.GetVote(ctx, 0)
	require.Equal(v.VoteResult, types.VoteResult_NO_TALLY)
	require.Equal(v.Voters, []string{Addr.String()})
}

func (suite *IntegrationTestSuite) TestProposeDisputeFromBond() {
	_, msgServer := suite.disputeKeeper()
	require := suite.Require()
	ctx := suite.ctx
	// k := suite.disputekeeper
	report, valAddr := suite.microReport()
	val, found := suite.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)
	bondedTokensBefore := val.GetBondedTokens()
	onePercent := bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))

	disputeFee := sdk.NewCoin("stake", onePercent)
	// slashAmount := disputeFee.Amount
	_, err := msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         sdk.AccAddress(valAddr).String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             disputeFee,
		PayFromBond:     true,
	})
	require.NoError(err)

	val, _ = suite.stakingKeeper.GetValidator(ctx, valAddr)
	// require.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(slashAmount).Sub(disputeFee.Amount))
	require.True(val.IsJailed())
	// jail time for a warning is zero seconds so unjailing should be immediate
	// TODO: have to unjail through the staking keeper, if no self delegation then validator can't unjail
	suite.mintTokens(sdk.AccAddress(valAddr))
	_, err = suite.stakingKeeper.Delegate(ctx, sdk.AccAddress(valAddr), sdk.NewInt(10), stakingtypes.Unbonded, val, true)
	require.NoError(err)
	err = suite.slashingKeeper.Unjail(ctx, valAddr)
	require.NoError(err)
	val, _ = suite.stakingKeeper.GetValidator(ctx, valAddr)
	require.False(val.IsJailed())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) microReport() (types.MicroReport, sdk.ValAddress) {
	val := suite.stakingKeeper.GetAllValidators(suite.ctx)[0]
	valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
	suite.Require().NoError(err)
	return types.MicroReport{
		Reporter:  sdk.AccAddress(valAddr).String(),
		Power:     val.GetConsensusPower(val.GetBondedTokens()),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}, valAddr

}
