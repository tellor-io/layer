package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *IntegrationTestSuite) TestFailedBondQueryLegacyReceipts() {
	fx := s.newReportedStake()
	ctx := s.Setup.Ctx
	s.Require().NoError(s.Setup.Disputekeeper.Dust.Set(ctx, math.ZeroInt()))
	for _, tc := range []struct {
		name     string
		origins  []int64
		shared   bool
		expected int64
	}{
		{"short receipt", []int64{333_333, 666_666}, false, 999_999},
		{"equal receipt", []int64{1_000_000}, false, 1_000_000},
		{"shared hash receipt", []int64{3_000_000}, true, 1_000_000},
	} {
		s.Run(tc.name, func() {
			payer := sample.AccAddressBytes()
			hash := []byte(tc.name)
			paid := math.ZeroInt()
			for i, amount := range tc.origins {
				s.delegate(payer, fx.vals[i], math.NewInt(amount))
				paid = paid.AddRaw(amount)
			}
			s.registerSelfReporter(payer)
			rk := s.Setup.Reporterkeeper
			s.Require().NoError(rk.FeefromReporterStake(ctx, payer, paid, hash, true))
			key := feePaidFromStakePayerKey(hash, payer)
			receipt, err := rk.FeePaidFromStake.Get(ctx, key)
			s.Require().NoError(err)
			s.Require().Equal(paid, receipt.Total)
			s.Require().Len(receipt.TokenOrigins, len(tc.origins))
			if tc.shared {
				// Pre-v6.1.6 stored one receipt under the bare dispute hash.
				s.Require().NoError(rk.FeePaidFromStake.Remove(ctx, key))
				key = hash
				s.Require().NoError(rk.FeePaidFromStake.Set(ctx, key, receipt))
			}
			dk := s.Setup.Disputekeeper
			id := dk.NextDisputeId(ctx)
			s.Require().NoError(dk.Disputes.Set(ctx, id, types.Dispute{DisputeId: id, DisputeStatus: types.Failed, HashId: hash}))
			payerKey := collections.Join(id, payer.Bytes())
			s.Require().NoError(dk.DisputeFeePayer.Set(ctx, payerKey, types.PayerInfo{Amount: math.NewInt(1_000_000), FromBond: true}))

			req := &types.QueryClaimableDisputeRewardsRequest{DisputeId: id, Address: payer.String()}
			q := keeper.NewQuerier(dk)
			resp, err := q.ClaimableDisputeRewards(ctx, req)
			s.Require().NoError(err)
			s.Equal(math.NewInt(tc.expected), resp.ClaimableAmount.FeeRefundAmount)
			moduleBefore := s.moduleBalance()
			msg := &types.MsgWithdrawFeeRefund{CallerAddress: payer.String(), PayerAddress: payer.String(), Id: id}
			server := keeper.NewMsgServerImpl(dk)
			_, err = server.WithdrawFeeRefund(ctx, msg)
			s.Require().NoError(err)
			bonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(ctx, payer)
			s.Require().NoError(err)
			s.Equal(math.NewInt(tc.expected), bonded.Add(s.unbondingBalance(payer)))
			s.Equal(math.NewInt(tc.expected), moduleBefore.Sub(s.moduleBalance()))
			_, err = rk.FeePaidFromStake.Get(ctx, key)
			s.ErrorIs(err, collections.ErrNotFound)
			_, err = dk.DisputeFeePayer.Get(ctx, payerKey)
			s.ErrorIs(err, collections.ErrNotFound)
			resp, err = q.ClaimableDisputeRewards(ctx, req)
			s.Require().NoError(err)
			s.Equal(math.ZeroInt(), resp.ClaimableAmount.FeeRefundAmount)
			_, err = server.WithdrawFeeRefund(ctx, msg)
			s.ErrorIs(err, collections.ErrNotFound)
		})
	}
}
