package integration_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/testutil/sample"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// MsgAddFeeToDispute and MsgProposeDispute can return a round-1 fee straight
// into bonded stake (see adjustDisputeFeeToCollectedSlash). The reporter ante
// decorator cannot project that return. These tests pin the guard remedy: both
// messages are keeper-direct acquisitions, so a transaction that also holds an
// ante-projected acquisition is rejected with ErrMixedStakeAcquisitionPaths.
//
// They run the production TrackStakeChangesDecorator, the instance app/ante.go
// installs ahead of the fee and signature decorators, against real staking,
// reporter and dispute keepers through runStakeAnte. A mixed transaction is
// refused before any signature or balance is read, so the only error asserted
// is the guard's own sentinel. Each leg is shown valid on its own: it passes
// the decorator, executes against real state, and stays inside every cap.

// fundingAnteScene is a round-1 dispute fixture with a real from-bond payer.
// A Minor dispute already escrowed 5% of the reported stake, so a Major
// dispute collects less than its quote and funding returns each payer's
// excess. bondPayer is a reporter whose own stake pays from-bond fees and
// user holds liquid loya for account payments and delegations.
type fundingAnteScene struct {
	fx        reportedStake
	quote     math.Int
	bondPayer sdk.AccAddress
	bondVal   sdk.ValAddress
	user      sdk.AccAddress
}

func (s *IntegrationTestSuite) newFundingAnteScene(bondStake math.Int) fundingAnteScene {
	fx := s.newReportedStake()
	minorPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(minorPayer, math.NewInt(300_000_000))
	_, quote := s.fundMinorThenMajor(fx, minorPayer)

	scene := fundingAnteScene{
		fx:        fx,
		quote:     quote,
		bondPayer: sample.AccAddressBytes(),
		bondVal:   fx.vals[1],
		user:      sample.AccAddressBytes(),
	}
	s.delegate(scene.bondPayer, scene.bondVal, bondStake)
	s.registerSelfReporter(scene.bondPayer)
	return scene
}

// fundingAnteExecute runs the messages through their real handlers, in order,
// against a throwaway cache of the current state and returns that cache. No
// ante runs here: the handlers do not enforce the concentration caps, which is
// why a transaction mixing them with a staking acquisition matters.
func (s *IntegrationTestSuite) fundingAnteExecute(msgs ...sdk.Msg) sdk.Context {
	cacheCtx, _ := s.Setup.Ctx.CacheContext()
	disputeServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for _, msg := range msgs {
		var err error
		switch msg := msg.(type) {
		case *disputetypes.MsgAddFeeToDispute:
			_, err = disputeServer.AddFeeToDispute(cacheCtx, msg)
		case *disputetypes.MsgProposeDispute:
			_, err = disputeServer.ProposeDispute(cacheCtx, msg)
		case *stakingtypes.MsgDelegate:
			_, err = stakingServer.Delegate(cacheCtx, msg)
		default:
			s.Require().FailNow("unsupported message", "%T", msg)
		}
		s.Require().NoError(err, "%T must execute against real state", msg)
	}
	return cacheCtx
}

// fundingAnteBondedReturn sums the stake the keeper delegated straight back to
// a delegator on a validator while ctx's messages ran.
func (s *IntegrationTestSuite) fundingAnteBondedReturn(ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress) math.Int {
	returned := math.ZeroInt()
	for _, event := range ctx.EventManager().Events() {
		if event.Type != "tokens_added_to_stake" {
			continue
		}
		attrs := make(map[string]string, len(event.Attributes))
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["delegator"] != delegator.String() || attrs["validator"] != validator.String() {
			continue
		}
		amount, ok := math.NewIntFromString(attrs["amount"])
		s.Require().True(ok, "tokens_added_to_stake amount %q is not an integer", attrs["amount"])
		returned = returned.Add(amount)
	}
	return returned
}

// fundingAnteValidatorExceedsCap reports whether the validator holds more than
// maxShare of bonded stake in ctx, the same strict comparison the caps use.
func (s *IntegrationTestSuite) fundingAnteValidatorExceedsCap(ctx sdk.Context, val sdk.ValAddress, maxShare math.LegacyDec) bool {
	validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, val)
	s.Require().NoError(err)
	totalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(ctx)
	s.Require().NoError(err)
	return validator.Tokens.ToLegacyDec().GT(maxShare.MulInt(totalBonded))
}

// requireFundingAnteStandalonePasses is the positive control: each message of
// the mix is admitted by the decorator by itself, and the funding message is
// not rejected merely for sharing a transaction with a non-acquisition message.
func (s *IntegrationTestSuite) requireFundingAnteStandalonePasses(funding, acquisition sdk.Msg) {
	s.Run("standalone_funding_passes", func() {
		s.Require().NoError(s.runStakeAnte(s.T(), funding))
	})
	s.Run("standalone_acquisition_passes", func() {
		s.Require().NoError(s.runStakeAnte(s.T(), acquisition))
	})
	s.Run("funding_with_non_acquisition_passes", func() {
		send := banktypes.NewMsgSend(sample.AccAddressBytes(), sample.AccAddressBytes(), sdk.NewCoins(sdk.NewInt64Coin(s.Setup.Denom, 1)))
		s.Require().NoError(s.runStakeAnte(s.T(), funding, send))
	})
}

// requireFundingAnteMixRejected asserts the transaction-wide guard rejects the
// funding message together with the staking acquisition in either order and
// with either one wrapped in authz.
func (s *IntegrationTestSuite) requireFundingAnteMixRejected(funding, acquisition sdk.Msg) {
	grantee := sample.AccAddressBytes()
	wrap := func(msg sdk.Msg) sdk.Msg {
		exec := authz.NewMsgExec(grantee, []sdk.Msg{msg})
		return &exec
	}
	cases := []struct {
		name string
		msgs []sdk.Msg
	}{
		{name: "funding_then_acquisition", msgs: []sdk.Msg{funding, acquisition}},
		{name: "acquisition_then_funding", msgs: []sdk.Msg{acquisition, funding}},
		{name: "authz_funding_then_acquisition", msgs: []sdk.Msg{wrap(funding), acquisition}},
		{name: "funding_then_authz_acquisition", msgs: []sdk.Msg{funding, wrap(acquisition)}},
	}
	for _, tc := range cases {
		s.Run("mixed_"+tc.name+"_is_rejected", func() {
			err := s.runStakeAnte(s.T(), tc.msgs...)
			s.Require().ErrorIs(err, reportertypes.ErrMixedStakeAcquisitionPaths,
				"%T mixed with %T must hit the keeper-direct versus ante-projected guard", funding, acquisition)
		})
	}
}

// TestDisputeFundingAnteAddFeeMixedWithStakingAcquisition covers the final
// account top-up that completes a dispute a from-bond payer funded in part. The
// top-up returns that payer's excess to its bond and then a delegation in the
// same transaction can spend the validator headroom the refund already used.
//
// The cap is set so the validator has exactly `headroom` after the escrow has
// shrunk the bonded total. The refund fits inside it and so does the delegation
// on its own, but together they do not. Ante projects only the delegation, so
// without the guard it admits a transaction whose execution breaches the cap.
func (s *IntegrationTestSuite) TestDisputeFundingAnteAddFeeMixedWithStakingAcquisition() {
	scene := s.newFundingAnteScene(math.NewInt(50_000_000))
	denom := s.Setup.Denom

	// The bond payer funds 40% and leaves the Major dispute in Prevote. The
	// user's account top-up completes it and is the message under test.
	bondShare := scene.quote.MulRaw(40).QuoRaw(100)
	topUpAmount := scene.quote.Sub(bondShare)
	partial := s.proposeDispute(scene.fx.report, disputetypes.Major, scene.bondPayer, bondShare, true)
	s.Require().Equal(disputetypes.Prevote, partial.DisputeStatus)
	s.Require().True(s.payerInfo(partial.DisputeId, scene.bondPayer).FromBond)

	s.Setup.MintTokens(scene.user, topUpAmount.Add(scene.quote))
	topUp := &disputetypes.MsgAddFeeToDispute{
		Creator:     scene.user.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(denom, topUpAmount),
		PayFromBond: false,
	}
	s.setStakeTrackerToCurrentBonded()

	// Run the top-up once, with no validator cap, to learn what it returns to
	// the prior payer's bond and where the validator and the bonded total stand
	// when that refund is bonded.
	trial := s.fundingAnteExecute(topUp)
	completed, err := s.Setup.Disputekeeper.Disputes.Get(trial, partial.DisputeId)
	s.Require().NoError(err)
	s.Require().Equal(disputetypes.Voting, completed.DisputeStatus)
	s.Require().True(completed.SlashAmount.LT(scene.quote), "the Minor dispute must leave less than the quote to collect")
	refund := s.fundingAnteBondedReturn(trial, scene.bondPayer, scene.bondVal)
	s.Require().True(refund.IsPositive(), "the completing top-up must return the prior payer's excess to bonded stake")
	trialValidator, err := s.Setup.Stakingkeeper.GetValidator(trial, scene.bondVal)
	s.Require().NoError(err)
	trialTotalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(trial)
	s.Require().NoError(err)
	validatorBeforeRefund := trialValidator.Tokens.Sub(refund)
	totalBeforeRefund := trialTotalBonded.Sub(refund)

	headroom := refund.MulRaw(3)
	delegation := headroom.Sub(refund.QuoRaw(2))
	maxShare := math.LegacyNewDecFromInt(validatorBeforeRefund.Add(headroom)).
		Quo(math.LegacyNewDecFromInt(totalBeforeRefund.Add(headroom)))
	params, err := s.Setup.Reporterkeeper.Params.Get(s.Setup.Ctx)
	s.Require().NoError(err)
	params.MaxValidatorPowerShare = maxShare
	s.Require().NoError(s.Setup.Reporterkeeper.Params.Set(s.Setup.Ctx, params))
	delegateMsg := stakingtypes.NewMsgDelegate(scene.user.String(), scene.bondVal.String(), sdk.NewCoin(denom, delegation))

	s.Run("funding_message_bonds_the_refund_inside_the_cap", func() {
		executed := s.fundingAnteExecute(topUp)
		bonded := s.fundingAnteBondedReturn(executed, scene.bondPayer, scene.bondVal)
		s.Require().Equal(refund.String(), bonded.String())
		s.Require().False(s.fundingAnteValidatorExceedsCap(executed, scene.bondVal, maxShare))
	})
	s.Run("delegation_alone_stays_inside_the_cap", func() {
		executed := s.fundingAnteExecute(delegateMsg)
		s.Require().False(s.fundingAnteValidatorExceedsCap(executed, scene.bondVal, maxShare))
	})
	s.Run("combined_execution_exceeds_the_cap", func() {
		executed := s.fundingAnteExecute(topUp, delegateMsg)
		s.Require().True(s.fundingAnteValidatorExceedsCap(executed, scene.bondVal, maxShare),
			"the refund and the delegation each fit the validator cap alone and breach it together")
	})

	s.requireFundingAnteStandalonePasses(topUp, delegateMsg)
	s.requireFundingAnteMixRejected(topUp, delegateMsg)
}

// TestDisputeFundingAnteProposeDisputeMixedWithStakingAcquisition covers a
// from-bond proposer who pays the whole Major quote. The Minor dispute left
// less than the quote to collect, so the same message returns the excess to the
// proposer's bond. The proposer's net bonded stake still falls, so this does
// not show a cap breach. It shows that the refund bonds inside the message,
// ahead of anything ante projected, which is all the guard keys on.
func (s *IntegrationTestSuite) TestDisputeFundingAnteProposeDisputeMixedWithStakingAcquisition() {
	scene := s.newFundingAnteScene(math.NewInt(150_000_000))
	denom := s.Setup.Denom
	report := scene.fx.report
	disputeID := s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx)

	propose := &disputetypes.MsgProposeDispute{
		Creator:          scene.bondPayer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		DisputeCategory:  disputetypes.Major,
		Fee:              sdk.NewCoin(denom, scene.quote),
		PayFromBond:      true,
	}
	delegation := math.NewInt(5_000_000)
	s.Setup.MintTokens(scene.user, delegation)
	delegateMsg := stakingtypes.NewMsgDelegate(scene.user.String(), scene.bondVal.String(), sdk.NewCoin(denom, delegation))
	s.setStakeTrackerToCurrentBonded()

	s.Run("proposal_returns_the_excess_to_bond_itself", func() {
		executed := s.fundingAnteExecute(propose)
		created, err := s.Setup.Disputekeeper.Disputes.Get(executed, disputeID)
		s.Require().NoError(err)
		s.Require().Equal(disputetypes.Voting, created.DisputeStatus)
		s.Require().True(created.SlashAmount.LT(scene.quote), "the Minor dispute must leave less than the quote to collect")
		refund := s.fundingAnteBondedReturn(executed, scene.bondPayer, scene.bondVal)
		s.Require().True(refund.IsPositive(), "the proposal must return the proposer's excess to bonded stake")
		s.fundingAnteExecute(delegateMsg)
	})

	s.requireFundingAnteStandalonePasses(propose, delegateMsg)
	s.requireFundingAnteMixRejected(propose, delegateMsg)
}
