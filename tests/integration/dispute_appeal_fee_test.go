package integration_test

import (
	"encoding/hex"
	"strconv"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) TestAppealExactOfferFromRetainedFee() {
	for _, test := range []struct {
		retained int64
		prices   [4]int64
	}{
		{15, [4]int64{1, 3, 6, 12}},
		{19, [4]int64{1, 3, 7, 15}},
		{20, [4]int64{2, 4, 8, 16}},
	} {
		s.Run(strconv.FormatInt(test.retained, 10), func() {
			s.SetupTest()
			fx := s.newReportedStake()
			s.leaveReporterSlashable(fx, math.NewInt(test.retained))
			payer := sample.AccAddressBytes()
			s.Setup.MintTokens(payer, fx.stake.AddRaw(50))
			current := s.proposeDispute(fx.report, types.Major, payer, fx.stake, false)
			s.Require().Equal(math.NewInt(test.retained), current.DisputeFee)
			firstID := current.DisputeId
			for i, price := range test.prices {
				before := s.moduleBalance()
				current.DisputeStatus = types.Unresolved
				s.Require().NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, current.DisputeId, current))
				appeal := s.proposeDispute(fx.report, types.Major, payer, math.NewInt(price), false)
				s.Equal(uint64(i+2), appeal.DisputeRound)
				s.Equal(current.FeeTotal.AddRaw(price), appeal.FeeTotal)
				s.Equal(current.BurnAmount.AddRaw(price), appeal.BurnAmount)
				s.Equal(before.AddRaw(price), s.moduleBalance())
				s.Equal(fx.stake, appeal.SlashAmount)
				s.Equal(current.DisputeFee, appeal.DisputeFee)
				current = appeal
			}
			s.executeVote(current, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST)
			ctx := s.Setup.Ctx
			query := keeper.NewQuerier(s.Setup.Disputekeeper)
			expected := fx.stake.SubRaw(test.retained)
			req := &types.QueryClaimableDisputeRewardsRequest{DisputeId: firstID, Address: payer.String()}
			old, err := query.ClaimableDisputeRewards(ctx, req)
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), old.ClaimableAmount.FeeRefundAmount)
			req.DisputeId = current.DisputeId
			final, err := query.ClaimableDisputeRewards(ctx, req)
			s.Require().NoError(err)
			s.Equal(expected, final.ClaimableAmount.FeeRefundAmount)

			server := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
			claim := &types.MsgWithdrawFeeRefund{Id: current.DisputeId, CallerAddress: payer.String(), PayerAddress: payer.String()}
			balance := s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.Require().NoError(err)
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.ErrorIs(err, collections.ErrNotFound)
			s.Equal(balance.Add(expected), s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount)
		})
	}
}

func (s *IntegrationTestSuite) TestEmptyRoundAppealAndClaim() {
	for _, outcome := range []types.VoteResult{types.VoteResult_INVALID, types.VoteResult_SUPPORT} {
		s.Run(outcome.String(), func() {
			s.SetupTest()
			fx := s.newReportedStake()
			ctx := s.Setup.Ctx
			fx.report.Power = 0
			s.Require().NoError(s.Setup.Oraclekeeper.Reports.Set(ctx, collections.Join3(fx.report.QueryId, fx.reporter.Bytes(), fx.report.MetaId), fx.report))
			s.Require().NoError(s.Setup.Reporterkeeper.ReportByBlock.Set(ctx, collections.Join3(fx.reporter.Bytes(), fx.report.BlockNumber, fx.report.QueryId), reportertypes.DelegationsAmounts{Total: math.ZeroInt()}))

			payer := sample.AccAddressBytes()
			s.Setup.MintTokens(payer, math.OneInt())
			server := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
			_, err := server.ProposeDispute(ctx, &types.MsgProposeDispute{
				Creator: payer.String(), DisputedReporter: fx.report.Reporter, ReportMetaId: fx.report.MetaId,
				ReportQueryId: hex.EncodeToString(fx.report.QueryId), DisputeCategory: types.Major,
				Fee: sdk.NewCoin(s.Setup.Denom, math.OneInt()), PayFromBond: true,
			})
			s.Require().ErrorIs(err, types.ErrMinimumTRBrequired)
			first := s.proposeDispute(fx.report, types.Major, payer, layertypes.OnePercent, true)
			s.Equal(math.ZeroInt(), first.SlashAmount)
			s.Equal(math.ZeroInt(), first.DisputeFee)
			s.Equal(math.ZeroInt(), s.payerInfo(first.DisputeId, payer).Amount)
			receipt, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, first.HashId, payer)
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), receipt.Total)
			principal, err := s.Setup.Reporterkeeper.DisputedDelegationTotal(ctx, first.HashId)
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), principal)
			first.DisputeStatus = types.Unresolved
			s.Require().NoError(s.Setup.Disputekeeper.Disputes.Set(ctx, first.DisputeId, first))

			before := s.moduleBalance()
			final := s.proposeDispute(fx.report, types.Major, payer, math.OneInt(), false)
			s.Equal(math.OneInt(), final.BurnAmount)
			s.Equal(math.OneInt(), final.FeeTotal)
			s.Equal(before.AddRaw(1), s.moduleBalance())
			s.executeVote(final, outcome)
			ctx = s.Setup.Ctx
			s.Equal(before, s.moduleBalance())
			claim := &types.MsgWithdrawFeeRefund{CallerAddress: fx.reporter.String(), PayerAddress: payer.String(), Id: final.DisputeId}
			query, err := keeper.NewQuerier(s.Setup.Disputekeeper).ClaimableDisputeRewards(ctx, &types.QueryClaimableDisputeRewardsRequest{Address: payer.String(), DisputeId: final.DisputeId})
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), query.ClaimableAmount.FeeRefundAmount)
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.Require().NoError(err)
			_, err = server.WithdrawFeeRefund(ctx, claim)
			s.ErrorIs(err, collections.ErrNotFound)
			_, err = s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, first.HashId, payer)
			s.ErrorIs(err, collections.ErrNotFound)
			_, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(ctx, collections.Join(first.DisputeId, payer.Bytes()))
			s.ErrorIs(err, collections.ErrNotFound)
			dust, err := s.Setup.Disputekeeper.Dust.Get(ctx)
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), dust)
			s.Equal(before, s.moduleBalance())
			s.Equal(math.ZeroInt(), s.Setup.Bankkeeper.GetBalance(ctx, payer, s.Setup.Denom).Amount)
		})
	}
}
