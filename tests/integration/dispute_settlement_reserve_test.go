package integration_test

import (
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	dispute "github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestSettlementPreservesOtherPayersAndVoterReserve() {
	for _, result := range []types.VoteResult{types.VoteResult_AGAINST, types.VoteResult_INVALID, types.VoteResult_SUPPORT} {
		s.Run(result.String(), func() {
			s.SetupTest()
			fx := s.newExactSettlementFixture()
			report := fx.dispute.InitialEvidence
			s.Require().NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, fx.reporter.Bytes(), report.MetaId), report))
			payer := sample.AccAddressBytes()
			pledge := layertypes.OnePercent
			s.Setup.MintTokens(payer, pledge)
			open := s.proposeDispute(report, types.Minor, payer, pledge, false)
			s.Require().Equal(types.Prevote, open.DisputeStatus)

			ctx := s.Setup.Ctx
			s.Require().NoError(s.Setup.Disputekeeper.VoteCountsByGroup.Set(ctx, fx.dispute.DisputeId, types.StakeholderVoteCounts{Users: types.VoteCounts{Support: 1}}))
			supply := s.Setup.Bankkeeper.GetSupply(ctx, s.Setup.Denom).Amount
			s.executeVote(fx.dispute, result)
			ctx = s.Setup.Ctx
			executed, err := s.Setup.Disputekeeper.Disputes.Get(ctx, fx.dispute.DisputeId)
			s.Require().NoError(err)
			s.Equal(math.OneInt(), executed.VoterReward)
			s.Equal(supply.SubRaw(2), s.Setup.Bankkeeper.GetSupply(ctx, s.Setup.Denom).Amount)
			s.Require().NoError(s.settlementClaim(fx, fx.cash))
			s.Require().NoError(s.settlementClaim(fx, fx.bond))
			s.Equal(pledge.AddRaw(1), s.moduleBalance())

			ctx = ctx.WithBlockTime(open.DisputeEndTime.Add(time.Second))
			s.Require().NoError(dispute.CheckOpenDisputesForExpiration(ctx, s.Setup.Disputekeeper))
			_, err = keeper.NewMsgServerImpl(s.Setup.Disputekeeper).WithdrawFeeRefund(ctx, &types.MsgWithdrawFeeRefund{
				Id: open.DisputeId, PayerAddress: payer.String(), CallerAddress: payer.String(),
			})
			s.Require().NoError(err)
			s.Equal(pledge, s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount)
			s.Equal(math.OneInt(), s.moduleBalance())
		})
	}
}

// The cash leg runs first. A full queue on the subsequent SUPPORT principal
// leg must roll back that cash transfer along with the rest of the claim.
func (s *IntegrationTestSuite) TestSupportPrincipalFailureRollsBackCashRefund() {
	fx := s.newExactSettlementFixture()
	s.executeVote(fx.dispute, types.VoteResult_SUPPORT)
	ctx := s.Setup.Ctx
	maxEntries, err := s.Setup.Stakingkeeper.MaxEntries(ctx)
	s.Require().NoError(err)
	val := fx.vals[0]
	s.delegate(fx.cash, val, math.NewInt(int64(maxEntries)+1))
	staking := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for range maxEntries {
		ctx = ctx.WithBlockTime(ctx.BlockTime().Add(time.Second)).WithBlockHeight(ctx.BlockHeight() + 1)
		_, err = staking.Undelegate(ctx, stakingtypes.NewMsgUndelegate(fx.cash.String(), val.String(), sdk.NewCoin(s.Setup.Denom, math.OneInt())))
		s.Require().NoError(err)
	}
	params, err := s.Setup.Reporterkeeper.Params.Get(ctx)
	s.Require().NoError(err)
	params.MaxValidatorPowerShare = math.LegacySmallestDec()
	s.Require().NoError(s.Setup.Reporterkeeper.Params.Set(ctx, params))
	req := &types.QueryClaimableDisputeRewardsRequest{DisputeId: fx.dispute.DisputeId, Address: fx.cash.String()}
	q := keeper.NewQuerier(s.Setup.Disputekeeper)
	before, err := q.ClaimableDisputeRewards(ctx, req)
	s.Require().NoError(err)
	s.Equal(math.NewInt(109), before.ClaimableAmount.FeeRefundAmount)

	balance := s.Setup.Bankkeeper.GetBalance(ctx, fx.cash, s.Setup.Denom).Amount
	cached, _ := ctx.CacheContext()
	_, err = keeper.NewMsgServerImpl(s.Setup.Disputekeeper).WithdrawFeeRefund(cached, &types.MsgWithdrawFeeRefund{
		Id: fx.dispute.DisputeId, PayerAddress: fx.cash.String(), CallerAddress: fx.cash.String(),
	})
	s.Require().ErrorIs(err, stakingtypes.ErrMaxUnbondingDelegationEntries)
	s.Equal(balance.AddRaw(67), s.Setup.Bankkeeper.GetBalance(cached, fx.cash, s.Setup.Denom).Amount)
	s.Equal(balance, s.Setup.Bankkeeper.GetBalance(ctx, fx.cash, s.Setup.Denom).Amount)
	after, err := q.ClaimableDisputeRewards(ctx, req)
	s.Require().NoError(err)
	s.Equal(before.ClaimableAmount.FeeRefundAmount, after.ClaimableAmount.FeeRefundAmount)
}
