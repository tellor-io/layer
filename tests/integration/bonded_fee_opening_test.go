package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestBondedOpeningUsesBothSelectors() {
	s.checkBondedOpening(false)
}

func (s *IntegrationTestSuite) TestBondedPayerFinishesOwnPartialPayment() {
	s.checkBondedOpening(true)
}

func (s *IntegrationTestSuite) checkBondedOpening(topUp bool) {
	r := s.Require()
	fx := s.newReportedStake()
	ctx := s.Setup.Ctx
	payer, selector := sample.AccAddressBytes(), sample.AccAddressBytes()
	validatorAddr := fx.vals[1]
	s.registerSelfReporter(payer)
	r.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, selector, reportertypes.NewSelection(payer, 1)))
	s.delegate(payer, validatorAddr, math.NewInt(30_000_000))
	s.delegate(selector, validatorAddr, math.NewInt(31_000_000))
	validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, validatorAddr)
	r.NoError(err)
	// Slash one loya from the shared validator after both delegations exist.
	pub, err := validator.ConsPubKey()
	r.NoError(err)
	slashed, err := s.Setup.Stakingkeeper.Slash(ctx, sdk.GetConsAddress(pub), ctx.BlockHeight(), 1, math.LegacyNewDecWithPrec(1, 6))
	r.NoError(err)
	r.True(slashed.Equal(math.OneInt()))
	validator, err = s.Setup.Stakingkeeper.GetValidator(ctx, validatorAddr)
	r.NoError(err)
	for _, owner := range []sdk.AccAddress{payer, selector} {
		delegation, err := s.Setup.Stakingkeeper.GetDelegation(ctx, owner, validatorAddr)
		r.NoError(err)
		r.False(validator.TokensFromShares(delegation.Shares).IsInteger())
	}

	quote := fx.stake.QuoRaw(20) // Minor disputes cost 5% of report power.
	payment := quote
	if topUp {
		payment = quote.QuoRaw(2)
	}
	pool := s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	bondedBefore := s.Setup.Bankkeeper.GetBalance(ctx, pool, s.Setup.Denom).Amount
	disputeBefore := s.moduleBalance()
	dispute := s.proposeDispute(fx.report, disputetypes.Minor, payer, payment, true)
	assertReceipt := func(want math.Int) {
		tracker, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, dispute.HashId, payer)
		r.NoError(err)
		r.Equal(want, tracker.Total)
		r.Len(tracker.TokenOrigins, 2)
		var owners []sdk.AccAddress
		total := math.ZeroInt()
		for _, origin := range tracker.TokenOrigins {
			r.True(origin.Amount.IsPositive())
			r.Equal(validatorAddr.Bytes(), origin.ValidatorAddress)
			owners = append(owners, sdk.AccAddress(origin.DelegatorAddress))
			total = total.Add(origin.Amount)
		}
		r.ElementsMatch([]sdk.AccAddress{payer, selector}, owners)
		r.Equal(want, total)
	}
	assertReceipt(payment)
	if topUp {
		r.Equal(disputetypes.Prevote, dispute.DisputeStatus)
		r.Equal(payment, s.moduleBalance().Sub(disputeBefore))
		_, err := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper).AddFeeToDispute(ctx, &disputetypes.MsgAddFeeToDispute{
			Creator: payer.String(), DisputeId: dispute.DisputeId, Amount: sdk.NewCoin(s.Setup.Denom, quote.Sub(payment)), PayFromBond: true,
		})
		r.NoError(err)
		dispute, err = s.Setup.Disputekeeper.Disputes.Get(ctx, dispute.DisputeId)
		r.NoError(err)
		assertReceipt(quote)
	}
	r.Equal(disputetypes.Voting, dispute.DisputeStatus)
	r.Equal(quote, dispute.FeeTotal)
	r.Equal(quote, dispute.DisputeFee)
	r.Equal(disputetypes.PayerInfo{Amount: quote, FromBond: true}, s.payerInfo(dispute.DisputeId, payer))
	escrow, err := s.Setup.Reporterkeeper.DisputedDelegationTotal(ctx, dispute.HashId)
	r.NoError(err)
	r.Equal(quote, escrow)
	r.Equal(quote.Add(escrow), s.moduleBalance().Sub(disputeBefore))
	r.Equal(quote.Add(escrow), bondedBefore.Sub(s.Setup.Bankkeeper.GetBalance(ctx, pool, s.Setup.Denom).Amount))
}
