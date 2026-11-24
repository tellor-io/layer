package integration_test

import (
	"sort"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type exactSettlementFixture struct {
	dispute  disputetypes.Dispute
	cash     sdk.AccAddress
	bond     sdk.AccAddress
	reporter sdk.AccAddress
	vals     []sdk.ValAddress
}

func (s *IntegrationTestSuite) newExactSettlementFixture() exactSettlementFixture {
	cashAmount, bondAmount := math.NewInt(70), math.NewInt(30)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(5).WithBlockTime(time.Now())
	_, vals, _ := s.createValidatorAccs([]uint64{1000, 1000, 1000})
	cash := sample.AccAddressBytes()
	bond := sample.AccAddressBytes()
	reporter := sample.AccAddressBytes()

	// The reporter snapshot is 100 loya, but only 60 loya is escrowed. The
	// small amounts make every settlement leg observable without fixed-point
	// rounding in the origin fixture.
	reporterStake := math.NewInt(100)
	s.delegate(reporter, vals[0], reporterStake)
	s.registerSelfReporter(reporter)
	qID := []byte("settlement-query")
	height := uint64(s.Setup.Ctx.BlockHeight())
	s.NoError(s.Setup.Reporterkeeper.ReportByBlock.Set(s.Setup.Ctx, collections.Join3(reporter.Bytes(), height, qID), reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{{
			DelegatorAddress: reporter.Bytes(), ValidatorAddress: vals[0].Bytes(), Amount: reporterStake,
		}},
		Total: reporterStake,
	}))
	report := oracletypes.MicroReport{
		Reporter: reporter.String(), Power: 100, QueryId: qID, Value: testValue,
		Timestamp: s.Setup.Ctx.BlockTime().Add(-time.Hour), MetaId: 1, BlockNumber: height,
	}

	// The bond payer has two real positions. FeefromReporterStake records the
	// actual origins and transfers the 30-loya fee into the dispute module.
	s.delegate(bond, vals[1], math.NewInt(20))
	s.delegate(bond, vals[2], math.NewInt(10))
	s.registerSelfReporter(bond)
	hashID := []byte("settlement-100-60")
	s.NoError(s.Setup.Reporterkeeper.FeefromReporterStake(s.Setup.Ctx, bond, bondAmount, hashID, true))

	s.Setup.MintTokens(cash, cashAmount)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, cash, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, cashAmount))))
	s.NoError(s.Setup.Reporterkeeper.EscrowReporterStake(s.Setup.Ctx, reporter, report.Power, height, math.NewInt(60), reportertypes.StakeSnapshotRef{}, hashID))

	const disputeID = uint64(1)
	blockTime := s.Setup.Ctx.BlockTime()
	dispute := disputetypes.Dispute{
		HashId: hashID, DisputeId: disputeID, DisputeCategory: disputetypes.Major,
		DisputeStatus: disputetypes.Resolved, DisputeStartTime: blockTime.Add(-2 * time.Hour),
		DisputeEndTime: blockTime.Add(-time.Hour), DisputeRound: 1,
		SlashAmount: math.NewInt(100), DisputeFee: math.NewInt(60), FeeTotal: math.NewInt(100),
		BurnAmount: math.NewInt(3), InitialEvidence: report, PrevDisputeIds: []uint64{disputeID},
		Open: true, PendingExecution: true, BlockNumber: height,
	}
	s.NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, disputeID, dispute))
	s.NoError(s.Setup.Disputekeeper.Votes.Set(s.Setup.Ctx, disputeID, disputetypes.Vote{
		Id: disputeID, VoteStart: dispute.DisputeStartTime, VoteEnd: dispute.DisputeEndTime,
		VoteResult: disputetypes.VoteResult_NO_TALLY,
	}))
	s.NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(s.Setup.Ctx, collections.Join(disputeID, cash.Bytes()), disputetypes.PayerInfo{Amount: cashAmount}))
	s.NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(s.Setup.Ctx, collections.Join(disputeID, bond.Bytes()), disputetypes.PayerInfo{Amount: bondAmount, FromBond: true}))
	s.NoError(s.Setup.Disputekeeper.Dust.Set(s.Setup.Ctx, math.ZeroInt()))
	return exactSettlementFixture{dispute: dispute, cash: cash, bond: bond, reporter: reporter, vals: vals}
}

func (s *IntegrationTestSuite) settlementStake(del sdk.AccAddress) math.Int {
	total := math.ZeroInt()
	delegations, err := s.Setup.Stakingkeeper.GetAllDelegatorDelegations(s.Setup.Ctx, del)
	s.NoError(err)
	for _, delegation := range delegations {
		val, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		s.NoError(err)
		validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, val)
		s.NoError(err)
		total = total.Add(validator.TokensFromShares(delegation.Shares).TruncateInt())
	}
	return total.Add(s.unbondingBalance(del))
}

func (s *IntegrationTestSuite) settlementClaim(fx exactSettlementFixture, payer sdk.AccAddress) error {
	_, err := keeper.NewMsgServerImpl(s.Setup.Disputekeeper).WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: payer.String(), PayerAddress: payer.String(), Id: fx.dispute.DisputeId,
	})
	return err
}

func (s *IntegrationTestSuite) stakeEventAmounts(payer sdk.AccAddress) []math.Int {
	amounts := make([]math.Int, 0)
	for _, ev := range s.Setup.Ctx.EventManager().Events() {
		if ev.Type != "tokens_added_to_stake" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range ev.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["delegator"] != payer.String() {
			continue
		}
		amount, ok := math.NewIntFromString(attrs["amount"])
		if !ok {
			continue
		}
		amounts = append(amounts, amount)
	}
	sort.Slice(amounts, func(i, j int) bool { return amounts[i].LT(amounts[j]) })
	return amounts
}

func (s *IntegrationTestSuite) TestTwoPayerSettlementConservesFunds() {
	for _, expected := range []struct {
		result                        disputetypes.VoteResult
		cash, cashStake, bondedPayout int64
		reporter, firstClaimDust      int64
		bondCredits                   []math.Int
	}{
		{disputetypes.VoteResult_AGAINST, 28, 0, 12, 117, 0, []math.Int{math.NewInt(4), math.NewInt(8)}},
		{disputetypes.VoteResult_INVALID, 67, 0, 29, 60, 900_000, []math.Int{math.NewInt(10), math.NewInt(19)}},
		{disputetypes.VoteResult_SUPPORT, 67, 42, 47, 0, 900_000, []math.Int{math.NewInt(10), math.NewInt(18), math.NewInt(19)}},
	} {
		s.Run(expected.result.String(), func() {
			s.SetupTest()
			fx := s.newExactSettlementFixture()
			cashBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, fx.cash, s.Setup.Denom).Amount
			bondBefore := s.settlementStake(fx.bond)
			reporterBefore := s.settlementStake(fx.reporter)
			s.Require().Equal(math.NewInt(160), s.moduleBalance())
			s.executeVote(fx.dispute, expected.result)
			ctx := s.Setup.Ctx
			// With no eligible voter all three loya burn at execution.
			s.Equal(math.NewInt(157-expected.reporter), s.moduleBalance())
			s.True(math.NewInt(expected.reporter).Equal(s.settlementStake(fx.reporter).Sub(reporterBefore)))
			hasEscrow, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Has(ctx, fx.dispute.HashId)
			s.Require().NoError(err)
			s.Equal(expected.result == disputetypes.VoteResult_SUPPORT, hasEscrow)

			s.Require().NoError(s.settlementClaim(fx, fx.cash))
			s.Equal(math.NewInt(expected.cash), s.Setup.Bankkeeper.GetBalance(ctx, fx.cash, s.Setup.Denom).Amount.Sub(cashBefore))
			s.True(math.NewInt(expected.cashStake).Equal(s.settlementStake(fx.cash)))
			dust, err := s.Setup.Disputekeeper.Dust.Get(ctx)
			s.Require().NoError(err)
			s.Equal(math.NewInt(expected.firstClaimDust), dust)

			s.Require().NoError(s.settlementClaim(fx, fx.bond))
			s.Equal(math.NewInt(expected.bondedPayout), s.settlementStake(fx.bond).Sub(bondBefore))
			s.Equal(expected.bondCredits, s.stakeEventAmounts(fx.bond))
			dust, err = s.Setup.Disputekeeper.Dust.Get(ctx)
			s.Require().NoError(err)
			s.True(dust.IsZero())
			s.True(s.moduleBalance().IsZero())
			for _, payer := range []sdk.AccAddress{fx.cash, fx.bond} {
				has, err := s.Setup.Disputekeeper.DisputeFeePayer.Has(ctx, collections.Join(fx.dispute.DisputeId, payer.Bytes()))
				s.Require().NoError(err)
				s.False(has)
			}
		})
	}
}
