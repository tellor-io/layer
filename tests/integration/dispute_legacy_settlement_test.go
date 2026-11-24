package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Both historical layouts have SlashAmount == DisputeFee. A pre-resize SUPPORT
// round can still have a short escrow. A #1101 round can already have paid the excess.
func (s *IntegrationTestSuite) TestLegacyRoundOneSettlement() {
	for _, test := range []struct {
		name    string
		pushed  bool
		outcome types.VoteResult
	}{
		{"short reporter escrow", false, types.VoteResult_SUPPORT},
		{"previously refunded excess on support", true, types.VoteResult_SUPPORT},
		{"previously refunded excess on against", true, types.VoteResult_AGAINST},
	} {
		s.Run(test.name, func() {
			s.SetupTest()
			fx := s.newReportedStake()
			collected := math.NewInt(60)
			s.leaveReporterSlashable(fx, collected)
			payer := sample.AccAddressBytes()
			s.Setup.MintTokens(payer, fx.stake)
			d := s.proposeDispute(fx.report, types.Major, payer, fx.stake, false)
			s.Require().Equal(collected, d.DisputeFee)
			ctx := s.Setup.Ctx
			if test.pushed {
				excess := fx.stake.Sub(collected)
				s.Require().NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payer, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, excess))))
				d.SlashAmount = collected
				d.FeeTotal = collected
				s.Require().NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(d.DisputeId, payer.Bytes()), types.PayerInfo{Amount: collected}))
			} else {
				d.DisputeFee = fx.stake
				d.BurnAmount = fx.stake.QuoRaw(20)
			}
			s.Require().NoError(s.Setup.Disputekeeper.Disputes.Set(ctx, d.DisputeId, d))
			s.executeVote(d, test.outcome)
			ctx = s.Setup.Ctx
			query := &types.QueryClaimableDisputeRewardsRequest{Address: payer.String(), DisputeId: d.DisputeId}
			server := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
			claim := &types.MsgWithdrawFeeRefund{CallerAddress: payer.String(), PayerAddress: payer.String(), Id: d.DisputeId}
			resp, err := keeper.NewQuerier(s.Setup.Disputekeeper).ClaimableDisputeRewards(ctx, query)
			s.Require().NoError(err)
			balance := s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount
			if test.outcome == types.VoteResult_AGAINST {
				s.Equal(math.ZeroInt(), resp.ClaimableAmount.FeeRefundAmount)
				_, err = server.WithdrawFeeRefund(ctx, claim)
				s.EqualError(err, "invalid vote result")
				s.Equal(balance, s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount)
				return
			}
			fee := d.SlashAmount.Sub(d.SlashAmount.QuoRaw(20))
			s.Equal(fee.Add(collected), resp.ClaimableAmount.FeeRefundAmount)
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.Require().NoError(err)
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.ErrorIs(err, collections.ErrNotFound)
			s.Equal(balance.Add(fee), s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount)
			bonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, payer)
			s.Require().NoError(err)
			s.Equal(collected, bonded.Add(s.unbondingBalance(payer)))
		})
	}
}

func (s *IntegrationTestSuite) TestRoundedZeroBondedExcessSettlesOnce() {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(60))
	opener, payer := sample.AccAddressBytes(), sample.AccAddressBytes()
	opening := fx.stake.SubRaw(1)
	s.Setup.MintTokens(opener, opening)
	d := s.proposeDispute(fx.report, types.Major, opener, opening, false)
	s.delegate(payer, fx.vals[1], math.OneInt())
	s.registerSelfReporter(payer)
	server := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := server.AddFeeToDispute(s.Setup.Ctx, &types.MsgAddFeeToDispute{
		Creator: payer.String(), DisputeId: d.DisputeId, Amount: sdk.NewCoin(s.Setup.Denom, math.OneInt()), PayFromBond: true,
	})
	s.Require().NoError(err)
	d, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, d.DisputeId)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(60), d.DisputeFee)
	s.executeVote(d, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
	ctx := s.Setup.Ctx
	available := s.moduleBalance()
	resp, err := keeper.NewQuerier(s.Setup.Disputekeeper).ClaimableDisputeRewards(ctx, &types.QueryClaimableDisputeRewardsRequest{DisputeId: d.DisputeId, Address: payer.String()})
	s.Require().NoError(err)
	s.Equal(math.ZeroInt(), resp.ClaimableAmount.FeeRefundAmount)
	dust, err := s.Setup.Disputekeeper.Dust.Get(ctx)
	s.Require().NoError(err)
	s.Equal(math.ZeroInt(), dust)
	claim := &types.MsgWithdrawFeeRefund{Id: d.DisputeId, CallerAddress: opener.String(), PayerAddress: payer.String()}
	_, err = server.WithdrawFeeRefund(ctx, claim)
	s.Require().NoError(err)
	_, err = server.WithdrawFeeRefund(ctx, claim)
	s.ErrorIs(err, collections.ErrNotFound)
	_, err = s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, d.HashId, payer)
	s.ErrorIs(err, collections.ErrNotFound)
	dust, err = s.Setup.Disputekeeper.Dust.Get(ctx)
	s.Require().NoError(err)
	s.Equal(math.NewInt(999_999), dust) // 1 × (100,000,000 − 60) / 100,000,000 loya
	s.Equal(available, s.moduleBalance())
}

func (s *IntegrationTestSuite) TestAgainstExcessUsesHistoricalBondBacking() {
	fx := s.newExactSettlementFixture()
	ctx := s.Setup.Ctx
	receipt, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, fx.dispute.HashId, fx.bond)
	s.Require().NoError(err)
	// Model a bonded contribution recorded as 30 for which only 29 arrived.
	receipt.Total = receipt.Total.SubRaw(1)
	receipt.TokenOrigins[0].Amount = receipt.TokenOrigins[0].Amount.SubRaw(1)
	s.Require().NoError(s.Setup.Reporterkeeper.FeePaidFromStake.Set(ctx, feePaidFromStakePayerKey(fx.dispute.HashId, fx.bond), receipt))
	s.Require().NoError(s.Setup.Bankkeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, math.OneInt()))))
	s.executeVote(fx.dispute, types.VoteResult_AGAINST)
	ctx = s.Setup.Ctx
	req := &types.QueryClaimableDisputeRewardsRequest{DisputeId: fx.dispute.DisputeId, Address: fx.bond.String()}
	resp, err := keeper.NewQuerier(s.Setup.Disputekeeper).ClaimableDisputeRewards(ctx, req)
	s.Require().NoError(err)
	s.Equal(math.NewInt(11), resp.ClaimableAmount.FeeRefundAmount)
	before := s.moduleBalance()
	s.Require().NoError(s.settlementClaim(fx, fx.bond))
	s.Equal(math.NewInt(11), before.Sub(s.moduleBalance()))
	s.ErrorIs(s.settlementClaim(fx, fx.bond), collections.ErrNotFound)
	resp, err = keeper.NewQuerier(s.Setup.Disputekeeper).ClaimableDisputeRewards(ctx, req)
	s.Require().NoError(err)
	s.Equal(math.ZeroInt(), resp.ClaimableAmount.FeeRefundAmount)
}
