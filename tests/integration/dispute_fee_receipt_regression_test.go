package integration_test

// Regressions for finding 1 of the PR 1101 review and for the inherited reporter-win behavior A.
//
// Finding 1: a fee paid from bond is credited at the amount requested, although fewer coins reach
// the dispute module. Funding records must credit the coins received, keep the difference
// outstanding and fundable, and complete funding only once real receipts reach the slash.
//
// Behavior A: the reporter-win payout on AGAINST is sized by the recorded fee. It must be paid
// from the coins the dispute actually holds and must never reach into another dispute's fees.
//
// A bonded payment is cut into one piece per selector of the paying reporter, and each piece is
// truncated to whole loya. The fixture reporter has 80 selectors with 1,000,000 loya each. 1/80
// is 0.0125, exact in LegacyDec, so every selector pays floor(amount/80) and a request delivers
// 80*floor(amount/80) loya, which is (amount mod 80) less than asked:
//
//	39,999,999 requested reaches the dispute module as 39,999,920 (79 lost)
//	29,999,999 requested reaches the dispute module as 29,999,920 (79 lost)
//	40,000,079 requested reaches the dispute module as 40,000,000 (79 lost)
//
// Amounts are loya. The disputed reporter has power 100, so the major quote is 100,000,000.

import (
	"strconv"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// receiptRegSelectors is below the default limit of 100 selectors per reporter.
	receiptRegSelectors     = 80
	receiptRegSelectorStake = int64(1_000_000)
	// receiptRegBondOpening is the request that opens the reporter-win fixtures from bond.
	// It reaches the dispute module as 39,999,920.
	receiptRegBondOpening = int64(39_999_999)
)

// receiptRegLoya compares loya amounts by their decimal string. testify's Equal falls back to
// reflect.DeepEqual on the big.Int inside math.Int, which can tell apart a zero built by NewInt
// and a zero produced by arithmetic, and the string shows both amounts when the check fails.
func (s *IntegrationTestSuite) receiptRegLoya(want, got math.Int, msgAndArgs ...any) {
	s.Equal(want.String(), got.String(), msgAndArgs...)
}

// receiptRegFixtureLoya is receiptRegLoya for numbers the fixture arithmetic fixes in advance.
// A failure means the fixture no longer builds the state the test describes.
func (s *IntegrationTestSuite) receiptRegFixtureLoya(want, got math.Int, msgAndArgs ...any) {
	s.Require().Equal(want.String(), got.String(), msgAndArgs...)
}

func (s *IntegrationTestSuite) receiptRegBalance(addr sdk.AccAddress) math.Int {
	return s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, addr, s.Setup.Denom).Amount
}

func (s *IntegrationTestSuite) receiptRegSupply() math.Int {
	return s.Setup.Bankkeeper.GetSupply(s.Setup.Ctx, s.Setup.Denom).Amount
}

// receiptRegStake is the stake the addresses hold: their delegations plus their unbonding entries.
func (s *IntegrationTestSuite) receiptRegStake(addrs ...sdk.AccAddress) math.Int {
	total := math.ZeroInt()
	for _, addr := range addrs {
		bonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, addr)
		s.Require().NoError(err)
		unbonding, err := s.Setup.Stakingkeeper.GetDelegatorUnbonding(s.Setup.Ctx, addr)
		s.Require().NoError(err)
		total = total.Add(bonded).Add(unbonding)
	}
	return total
}

func (s *IntegrationTestSuite) receiptRegDispute(id uint64) disputetypes.Dispute {
	d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, id)
	s.Require().NoError(err)
	return d
}

// receiptRegEscrow is what escrow took from the disputed reporter.
func (s *IntegrationTestSuite) receiptRegEscrow(d disputetypes.Dispute) math.Int {
	snapshot, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, d.HashId)
	s.Require().NoError(err)
	return snapshot.Total
}

// receiptRegTracked is the stake a payer unbonded to pay its first-round fee.
func (s *IntegrationTestSuite) receiptRegTracked(d disputetypes.Dispute, payer sdk.AccAddress) reportertypes.DelegationsAmounts {
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, d.HashId, payer)
	s.Require().NoError(err)
	return tracked
}

// receiptRegDeliver runs a message the way BaseApp does: on a cache of the context that is
// written back only when the handler succeeds.
func (s *IntegrationTestSuite) receiptRegDeliver(handler func(ctx sdk.Context) error) error {
	cacheCtx, write := s.Setup.Ctx.CacheContext()
	if err := handler(cacheCtx); err != nil {
		return err
	}
	write()
	return nil
}

// receiptRegAddFee delivers a MsgAddFeeToDispute.
func (s *IntegrationTestSuite) receiptRegAddFee(disputeID uint64, payer sdk.AccAddress, amount math.Int, fromBond bool) error {
	return s.receiptRegDeliver(func(ctx sdk.Context) error {
		_, err := keeper.NewMsgServerImpl(s.Setup.Disputekeeper).AddFeeToDispute(ctx, &disputetypes.MsgAddFeeToDispute{
			Creator:     payer.String(),
			DisputeId:   disputeID,
			Amount:      sdk.NewCoin(s.Setup.Denom, amount),
			PayFromBond: fromBond,
		})
		return err
	})
}

// receiptRegBondPayer is a reporter whose stake is spread over many selectors, so that a fee paid
// from its bond is cut into that many truncated pieces.
type receiptRegBondPayer struct {
	reporter  sdk.AccAddress
	selectors []sdk.AccAddress
	stake     math.Int
}

// receiptRegNewBondPayer registers a reporter that is also the first of 80 selectors. The others
// join through MsgSelectReporter. Every selector delegates 1,000,000 loya to val.
func (s *IntegrationTestSuite) receiptRegNewBondPayer(val sdk.ValAddress) receiptRegBondPayer {
	stake := math.NewInt(receiptRegSelectorStake)
	reporter := sample.AccAddressBytes()
	s.delegate(reporter, val, stake)
	s.registerSelfReporter(reporter)
	payer := receiptRegBondPayer{
		reporter:  reporter,
		selectors: []sdk.AccAddress{reporter},
		stake:     stake.MulRaw(receiptRegSelectors),
	}
	reporterServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)
	for i := 1; i < receiptRegSelectors; i++ {
		selector := sample.AccAddressBytes()
		s.delegate(selector, val, stake)
		_, err := reporterServer.SelectReporter(s.Setup.Ctx, &reportertypes.MsgSelectReporter{
			SelectorAddress: selector.String(),
			ReporterAddress: reporter.String(),
		})
		s.Require().NoError(err)
		payer.selectors = append(payer.selectors, selector)
	}
	return payer
}

// receiptRegResized is the state a funded dispute must end in when the reporter had less stake
// than the quote, so the dispute keeps only what was collected.
type receiptRegResized struct {
	disputeID     uint64
	bond          receiptRegBondPayer
	account       sdk.AccAddress
	accountBefore math.Int
	moduleBase    math.Int
	// collected is what escrow took from the reporter. The dispute keeps that much fee.
	collected   math.Int
	keptBond    math.Int
	keptAccount math.Int
}

// receiptRegRequireResizedFunding checks a dispute opened for voting by a payment that completed
// funding with a collected slash below the quote.
func (s *IntegrationTestSuite) receiptRegRequireResizedFunding(w receiptRegResized) {
	d := s.receiptRegDispute(w.disputeID)
	s.Equal(disputetypes.Voting, d.DisputeStatus, "the outstanding amount completes funding")
	s.receiptRegFixtureLoya(w.collected, s.receiptRegEscrow(d), "fixture: escrow took the stake the reporter had left")
	s.receiptRegLoya(w.collected, d.SlashAmount, "the slash is resized to the collected stake")
	s.receiptRegLoya(w.collected, d.DisputeFee, "the dispute fee is resized to the collected stake")
	s.receiptRegLoya(w.collected, d.FeeTotal, "the fee total is resized to the collected stake")
	s.receiptRegLoya(w.collected.QuoRaw(20), d.BurnAmount, "the burn allocation is 5% of the retained fee")

	bondInfo := s.payerInfo(d.DisputeId, w.bond.reporter)
	s.True(bondInfo.FromBond)
	s.receiptRegLoya(w.keptBond, bondInfo.Amount, "bonded payer's share of the retained fee")
	s.receiptRegLoya(w.keptBond, s.receiptRegTracked(d, w.bond.reporter).Total, "bonded payer's tracker is the share still at risk")
	accountInfo := s.payerInfo(d.DisputeId, w.account)
	s.False(accountInfo.FromBond)
	s.receiptRegLoya(w.keptAccount, accountInfo.Amount, "account payer's share of the retained fee")
	s.receiptRegLoya(w.collected, bondInfo.Amount.Add(accountInfo.Amount), "shares add up to the retained fee")

	s.receiptRegLoya(w.moduleBase.Add(w.collected.MulRaw(2)), s.moduleBalance(), "the module holds the retained fee and the escrow, nothing else")
	s.receiptRegLoya(w.accountBefore.Sub(w.keptAccount), s.receiptRegBalance(w.account), "account payer is refunded what it paid beyond its share")
	s.receiptRegLoya(math.ZeroInt(), s.receiptRegBalance(w.bond.reporter), "a bonded payer is never refunded in liquid coins")
	s.receiptRegLoya(w.bond.stake.Sub(w.keptBond), s.receiptRegStake(w.bond.selectors...), "bonded payer's selectors hold all stake except the share at risk")
}

// TestBondedInitialPaymentCreditsCoinsReceivedWithinOldTolerance opens a major dispute with
// 39,999,999 from bond when the reporter has 99,999,990 left to escrow (10 short of the quote).
//
// Only 39,999,920 reaches the module, so the payer record and the fee total must be 39,999,920 and
// 60,000,080 is outstanding. An account payer covers that. Funding then completes with every
// quote coin present, the 10 loya shortfall is refunded in proportion, and the dispute keeps the
// 99,999,990 the reporter lost:
//
//	P: 39,999,920 * 99,999,990 / 100,000,000 = 39,999,916.000008 -> 39,999,916 (refund 4)
//	A: 60,000,080 * 99,999,990 / 100,000,000 = 60,000,073.999992 -> 60,000,074 (refund 6)
//
// P's 79 loya loss is inside the old 100 loya tolerance. Crediting the 39,999,999 requested left
// 60,000,001 outstanding, 99,999,921 coins on hand, and a dispute shrunk below the 99,999,990 it
// holds in escrow.
func (s *IntegrationTestSuite) TestBondedInitialPaymentCreditsCoinsReceivedWithinOldTolerance() {
	fx := s.newReportedStake()
	quote := fx.stake
	collected := quote.SubRaw(10)
	s.leaveReporterSlashable(fx, collected)
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, quote)
	accountBefore := s.receiptRegBalance(account)
	moduleBase := s.moduleBalance()

	requested := math.NewInt(39_999_999)
	d := s.proposeDispute(fx.report, disputetypes.Major, bond.reporter, requested, true)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	received := s.moduleBalance().Sub(moduleBase)
	s.receiptRegFixtureLoya(math.NewInt(39_999_920), received, "fixture: 80 equal selectors lose 79 loya of a 39,999,999 loya payment from bond")
	s.receiptRegFixtureLoya(received, s.receiptRegTracked(d, bond.reporter).Total, "fixture: the payer's tracker is the coins unbonded")

	info := s.payerInfo(d.DisputeId, bond.reporter)
	s.True(info.FromBond)
	s.receiptRegLoya(received, info.Amount, "the payer record must credit the coins received, not the %s requested", requested)
	s.receiptRegLoya(received, d.FeeTotal, "the fee total must count the coins received")
	outstanding := d.SlashAmount.Sub(d.FeeTotal)
	s.receiptRegLoya(quote.Sub(received), outstanding, "the coins lost to rounding stay outstanding on the funding record")

	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, outstanding, false), "paying the outstanding amount must complete funding")
	s.receiptRegRequireResizedFunding(receiptRegResized{
		disputeID:     d.DisputeId,
		bond:          bond,
		account:       account,
		accountBefore: accountBefore,
		moduleBase:    moduleBase,
		collected:     collected,
		keptBond:      math.NewInt(39_999_916),
		keptAccount:   math.NewInt(60_000_074),
	})
}

// TestBondedTopUpsCreditCoinsReceivedBeyondOldTolerance opens a major dispute with 20,000,000 from
// an account, then a bonded payer tops it up twice, with 39,999,999 and 29,999,999, when the
// reporter has 99,999,990 left to escrow (10 short of the quote).
//
// The top-ups reach the module as 39,999,920 and 29,999,920. After each, the payer record and the
// fee total must count the coins received (fee totals 59,999,920 and then 89,999,840) and the
// shortfall must stay outstanding. The two losses add up to 158 loya. Against the 10 loya the
// reporter is short that is a 148 loya excess, past the old tolerance of 100.
//
// The account payer then pays the 10,000,160 outstanding. Funding completes with 100,000,000
// coins present, the 10 loya shortfall is refunded in proportion, and the dispute keeps the
// 99,999,990 the reporter lost:
//
//	P: 69,999,840 * 99,999,990 / 100,000,000 = 69,999,833.000016 -> 69,999,833 (refund 7)
//	A: 30,000,160 * 99,999,990 / 100,000,000 = 30,000,156.999984 -> 30,000,157 (refund 3)
//
// Crediting the requested amounts left 10,000,002 outstanding and 99,999,842 coins on hand. The
// completing payment was rejected because the collected slash exceeds them by 148.
func (s *IntegrationTestSuite) TestBondedTopUpsCreditCoinsReceivedBeyondOldTolerance() {
	fx := s.newReportedStake()
	quote := fx.stake
	shortfall := int64(10)
	collected := quote.SubRaw(shortfall)
	s.leaveReporterSlashable(fx, collected)
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, quote)
	accountBefore := s.receiptRegBalance(account)
	moduleBase := s.moduleBalance()

	opening := math.NewInt(20_000_000)
	d := s.proposeDispute(fx.report, disputetypes.Major, account, opening, false)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)

	paid := math.ZeroInt()
	lost := math.ZeroInt()
	for _, topUp := range []struct{ requested, received int64 }{{39_999_999, 39_999_920}, {29_999_999, 29_999_920}} {
		requested := math.NewInt(topUp.requested)
		moduleBefore := s.moduleBalance()
		s.Require().NoError(s.receiptRegAddFee(d.DisputeId, bond.reporter, requested, true))
		received := s.moduleBalance().Sub(moduleBefore)
		s.receiptRegFixtureLoya(math.NewInt(topUp.received), received, "fixture: 80 equal selectors lose 79 loya of a %s loya payment from bond", requested)
		paid = paid.Add(received)
		lost = lost.Add(requested.Sub(received))

		d = s.receiptRegDispute(d.DisputeId)
		s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
		s.receiptRegFixtureLoya(paid, s.receiptRegTracked(d, bond.reporter).Total, "fixture: the payer's tracker is the coins unbonded")
		s.receiptRegLoya(paid, s.payerInfo(d.DisputeId, bond.reporter).Amount, "the payer record must credit the coins received, not the %s requested", requested)
		s.receiptRegLoya(opening.Add(paid), d.FeeTotal, "the fee total must count the coins received")
		s.receiptRegLoya(quote.Sub(opening).Sub(paid), d.SlashAmount.Sub(d.FeeTotal), "the coins lost to rounding stay outstanding on the funding record")
	}
	s.Require().True(lost.SubRaw(shortfall).GT(math.NewInt(100)), "fixture: the receipts must fall more than 100 loya short of the collected slash, lost %s", lost)

	outstanding := d.SlashAmount.Sub(d.FeeTotal)
	s.Require().True(outstanding.IsPositive())
	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, outstanding, false), "paying the outstanding amount shown on the funding record must complete funding")
	s.receiptRegRequireResizedFunding(receiptRegResized{
		disputeID:     d.DisputeId,
		bond:          bond,
		account:       account,
		accountBefore: accountBefore,
		moduleBase:    moduleBase,
		collected:     collected,
		keptBond:      math.NewInt(69_999_833),
		keptAccount:   math.NewInt(30_000_157),
	})
}

// TestBondedPaymentOfOutstandingLeavesRemainderFundable has an account open a major dispute with
// 59,999,921, which leaves 40,000,079 outstanding, and a bonded payer ask to cover all of it.
//
// Only 40,000,000 arrives, so 79 loya must stay outstanding and the dispute must stay in Prevote.
// The account payer then pays that 79 and funding completes with exactly the quote present. The
// reporter has the full quote to escrow, so nothing is resized or refunded.
//
// Crediting the 40,000,079 requested closed funding at once with 99,999,921 coins on hand and no
// way left to fund the missing 79.
func (s *IntegrationTestSuite) TestBondedPaymentOfOutstandingLeavesRemainderFundable() {
	fx := s.newReportedStake()
	quote := fx.stake
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, quote)
	accountBefore := s.receiptRegBalance(account)
	moduleBase := s.moduleBalance()

	opening := math.NewInt(59_999_921)
	d := s.proposeDispute(fx.report, disputetypes.Major, account, opening, false)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	outstanding := d.SlashAmount.Sub(d.FeeTotal)
	s.receiptRegFixtureLoya(math.NewInt(40_000_079), outstanding, "fixture: what the opening payment leaves outstanding")

	moduleBefore := s.moduleBalance()
	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, bond.reporter, outstanding, true))
	received := s.moduleBalance().Sub(moduleBefore)
	s.receiptRegFixtureLoya(math.NewInt(40_000_000), received, "fixture: 80 equal selectors lose 79 loya of a 40,000,079 loya payment from bond")

	d = s.receiptRegDispute(d.DisputeId)
	s.Equal(disputetypes.Prevote, d.DisputeStatus, "funding is not complete until the coins received reach the slash")
	s.receiptRegLoya(opening.Add(received), d.FeeTotal, "the fee total must count the coins received")
	s.receiptRegLoya(math.NewInt(79), d.SlashAmount.Sub(d.FeeTotal), "the coins lost to rounding stay outstanding on the funding record")
	s.receiptRegLoya(received, s.payerInfo(d.DisputeId, bond.reporter).Amount, "the payer record must credit the coins received, not the %s requested", outstanding)
	s.receiptRegFixtureLoya(received, s.receiptRegTracked(d, bond.reporter).Total, "fixture: the payer's tracker is the coins unbonded")

	remainder := d.SlashAmount.Sub(d.FeeTotal)
	s.Require().True(remainder.IsPositive(), "a remainder must be left to fund")
	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, remainder, false), "paying the remainder must complete funding")

	d = s.receiptRegDispute(d.DisputeId)
	s.Equal(disputetypes.Voting, d.DisputeStatus)
	s.receiptRegFixtureLoya(quote, s.receiptRegEscrow(d), "fixture: the reporter had the whole quote to escrow")
	s.receiptRegLoya(quote, d.SlashAmount, "a full escrow leaves the slash at the quote")
	s.receiptRegLoya(quote, d.DisputeFee, "a full escrow leaves the dispute fee at the quote")
	s.receiptRegLoya(quote, d.FeeTotal, "the coins received add up to the quote")
	s.receiptRegLoya(received, s.payerInfo(d.DisputeId, bond.reporter).Amount, "bonded payer's share")
	s.receiptRegLoya(received, s.receiptRegTracked(d, bond.reporter).Total, "bonded payer's tracker")
	s.receiptRegLoya(opening.Add(remainder), s.payerInfo(d.DisputeId, account).Amount, "account payer's share")
	s.receiptRegLoya(moduleBase.Add(quote.MulRaw(2)), s.moduleBalance(), "the module holds the quote as fee and the quote as escrow")
	s.receiptRegLoya(accountBefore.Sub(opening).Sub(remainder), s.receiptRegBalance(account), "account payer paid its share and no refund is due")
	s.receiptRegLoya(bond.stake.Sub(received), s.receiptRegStake(bond.selectors...), "bonded payer's selectors lost the coins received")
}

// TestLegacyNominalPrevoteCreditReopensShortfall starts from a real bonded payment of 39,999,999
// that reaches the module as 39,999,920, then rewrites only the dispute fee total and the bonded
// payer's record to the nominal 39,999,999 that Prevote funding used to credit. The tracker and
// the module's coins stay real.
//
// A liquid payer sends the nominal outstanding 60,000,001. The funding record must return to the
// coins actually held, so the dispute stays in Prevote with exactly 79 fundable and no escrow or
// vote. Paying that 79 completes funding with the full quote present.
func (s *IntegrationTestSuite) TestLegacyNominalPrevoteCreditReopensShortfall() {
	fx := s.newReportedStake()
	quote := fx.stake
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, quote)
	moduleBase := s.moduleBalance()

	requested := math.NewInt(receiptRegBondOpening)
	d := s.proposeDispute(fx.report, disputetypes.Major, bond.reporter, requested, true)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	received := s.moduleBalance().Sub(moduleBase)
	s.receiptRegFixtureLoya(math.NewInt(39_999_920), received, "fixture: 80 equal selectors lose 79 loya of a 39,999,999 loya payment from bond")

	d.FeeTotal = requested
	s.Require().NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, d.DisputeId, d))
	s.Require().NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(s.Setup.Ctx, collections.Join(d.DisputeId, bond.reporter.Bytes()), disputetypes.PayerInfo{Amount: requested, FromBond: true}))

	nominal := quote.Sub(requested)
	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, nominal, false))

	d = s.receiptRegDispute(d.DisputeId)
	s.Equal(disputetypes.Prevote, d.DisputeStatus, "funding is not complete until the coins held reach the slash")
	s.receiptRegLoya(math.NewInt(79), d.SlashAmount.Sub(d.FeeTotal), "the nominal credit's shortfall is fundable again")
	s.receiptRegLoya(received, s.payerInfo(d.DisputeId, bond.reporter).Amount, "the bonded payer's record returns to the coins received")
	_, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, d.HashId)
	s.Error(err, "escrow must not run before funding completes")
	_, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, d.DisputeId)
	s.Error(err, "voting must not start before funding completes")

	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, d.SlashAmount.Sub(d.FeeTotal), false))

	d = s.receiptRegDispute(d.DisputeId)
	s.Equal(disputetypes.Voting, d.DisputeStatus)
	s.receiptRegFixtureLoya(quote, s.receiptRegEscrow(d), "fixture: the reporter had the whole quote to escrow")
	s.receiptRegLoya(quote, d.FeeTotal, "the coins held add up to the quote")
	s.receiptRegLoya(moduleBase.Add(quote.MulRaw(2)), s.moduleBalance(), "the module holds the quote as fee and the quote as escrow")
}

// receiptRegOpenVotingMajor opens a major dispute for a reporter with the whole quote to escrow.
// A bonded payer opens it with the 39,999,999 it requests, then an account payer pays exactly the
// outstanding amount the funding record shows. Funding completes and the dispute is in Voting.
func (s *IntegrationTestSuite) receiptRegOpenVotingMajor(fx reportedStake, bond receiptRegBondPayer, account sdk.AccAddress) disputetypes.Dispute {
	d := s.proposeDispute(fx.report, disputetypes.Major, bond.reporter, math.NewInt(receiptRegBondOpening), true)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	s.Require().NoError(s.receiptRegAddFee(d.DisputeId, account, d.SlashAmount.Sub(d.FeeTotal), false), "paying the outstanding amount must complete funding")
	d = s.receiptRegDispute(d.DisputeId)
	s.Require().Equal(disputetypes.Voting, d.DisputeStatus)
	return d
}

// receiptRegOpenLegacyVotingMajor builds the state of a major dispute that was fully funded before
// receipts were credited. This is a fixture, not current behavior: current funding credits the
// coins received and cannot reach this state.
//
// The bonded payment is real, so the coins and the payer's tracker come from real share rounding:
// 39,999,920 reach the module for the 39,999,999 requested. The records are then written as the
// earlier funding path wrote them: the bonded payer at 39,999,999 and an account payer at the
// 60,000,001 that completed the quote, with the account's coins really moved. The dispute is
// written with the quote as slash, fee and fee total and the 5% allocation of it, and the
// completion is replayed as that path did, with the escrow, voting and a vote start. The dispute
// is recorded at 100,000,000 but the module holds fee coins for only 99,999,921.
func (s *IntegrationTestSuite) receiptRegOpenLegacyVotingMajor(fx reportedStake, bond receiptRegBondPayer, account sdk.AccAddress) disputetypes.Dispute {
	ctx := s.Setup.Ctx
	quote := fx.stake
	requested := math.NewInt(receiptRegBondOpening)
	remainder := quote.Sub(requested)

	d := s.proposeDispute(fx.report, disputetypes.Major, bond.reporter, requested, true)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	s.Require().NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(d.DisputeId, bond.reporter.Bytes()), disputetypes.PayerInfo{Amount: requested, FromBond: true}))
	s.Require().NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, account, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, remainder))))
	s.Require().NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(d.DisputeId, account.Bytes()), disputetypes.PayerInfo{Amount: remainder, FromBond: false}))

	d.SlashAmount = quote
	d.DisputeFee = quote
	d.FeeTotal = quote
	d.BurnAmount = quote.QuoRaw(20)
	s.Require().NoError(s.Setup.Disputekeeper.SlashAndJailReporter(ctx, d.InitialEvidence, d.DisputeCategory, d.HashId))
	d.DisputeEndTime = ctx.BlockTime().Add(keeper.THREE_DAYS)
	d.DisputeStatus = disputetypes.Voting
	s.Require().NoError(s.Setup.Disputekeeper.SetStartVote(ctx, d.DisputeId))
	s.Require().NoError(s.Setup.Disputekeeper.Disputes.Set(ctx, d.DisputeId, d))
	return s.receiptRegDispute(d.DisputeId)
}

// receiptRegSettleAgainst has the voters vote AGAINST and lets the dispute BeginBlocker tally and
// execute the vote, as the chain does. The team alone is short of quorum, so the round is
// tallied once voting and the appeal window have ended. With a tipper the team and the tipper
// reach quorum and the round is executed at once.
//
// BeginBlocker drops an execution that fails, so the result is whether the vote was executed and
// the reason BeginBlocker reported when it was not.
func (s *IntegrationTestSuite) receiptRegSettleAgainst(d disputetypes.Dispute, tipper sdk.AccAddress) (executed bool, failure string) {
	team, err := s.Setup.Disputekeeper.GetTeamAddress(s.Setup.Ctx)
	s.Require().NoError(err)
	// The dispute EndBlocker records this snapshot for a dispute opened in the block.
	s.Require().NoError(s.Setup.Disputekeeper.SetBlockInfo(s.Setup.Ctx, d.HashId))
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	voters := []sdk.AccAddress{team}
	if tipper != nil {
		voters = append(voters, tipper)
	}
	for _, voter := range voters {
		_, err := msgServer.Vote(s.Setup.Ctx, &disputetypes.MsgVote{Voter: voter.String(), Id: d.DisputeId, Vote: disputetypes.VoteEnum_VOTE_AGAINST})
		s.Require().NoError(err)
	}
	if tipper == nil {
		s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(d.DisputeEndTime.Add(time.Second))
	}
	s.Require().NoError(dispute.BeginBlocker(s.Setup.Ctx, s.Setup.Disputekeeper))

	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, d.DisputeId)
	s.Require().NoError(err)
	return vote.Executed, s.receiptRegExecutionFailure(d.DisputeId)
}

// receiptRegExecutionFailure is the reason BeginBlocker gave for dropping the execution of a
// dispute, or "" when it reported none.
func (s *IntegrationTestSuite) receiptRegExecutionFailure(id uint64) string {
	want := strconv.FormatUint(id, 10)
	for _, ev := range s.Setup.Ctx.EventManager().Events() {
		if ev.Type != "dispute_execution_failed" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range ev.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["dispute_id"] == want {
			return attrs["reason"]
		}
	}
	return ""
}

// TestReporterWinPaysEscrowAndReceivedFeeWhenNoVoterQualifies settles an AGAINST vote on a major
// dispute that holds only its own coins. The reporter has the whole quote to escrow, so funding
// never resizes the dispute. A bonded payer requests 39,999,999 and an account payer covers the
// outstanding amount shown on the funding record. Both are credited for the coins that arrived.
//
// Nobody but the team votes, so no voter qualifies and the whole 5,000,000 allocation is burned.
// The winner must receive exactly the escrow plus the fee the dispute holds less the allocation,
// 100,000,000 + 100,000,000 - 5,000,000 = 195,000,000, and the module must end with nothing.
//
// A dispute that credited the 39,999,999 requested holds 99,999,921 of fee coins. It owes
// 200,000,000 in all, so the settlement overdraws the module by 79 loya and BeginBlocker cannot
// execute the vote.
func (s *IntegrationTestSuite) TestReporterWinPaysEscrowAndReceivedFeeWhenNoVoterQualifies() {
	fx := s.newReportedStake()
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, fx.stake)
	moduleBase := s.moduleBalance()

	d := s.receiptRegOpenVotingMajor(fx, bond, account)
	escrow := s.receiptRegEscrow(d)
	s.receiptRegFixtureLoya(fx.stake, escrow, "fixture: the reporter had the whole quote to escrow")
	funded := s.moduleBalance().Sub(moduleBase).Sub(escrow)
	s.receiptRegFixtureLoya(math.ZeroInt(), s.receiptRegStake(fx.reporter), "fixture: the reporter's whole stake is in escrow")
	supplyBefore := s.receiptRegSupply()

	executed, failure := s.receiptRegSettleAgainst(d, nil)
	s.Require().True(executed, "the reporter-win settlement must execute from the coins the dispute holds: %s", failure)

	settled := s.receiptRegDispute(d.DisputeId)
	s.receiptRegLoya(escrow.Add(funded).Sub(settled.BurnAmount), s.receiptRegStake(fx.reporter), "the winner receives the escrow plus the funded fee less the allocation")
	s.receiptRegLoya(settled.BurnAmount, supplyBefore.Sub(s.receiptRegSupply()), "no voter qualifies, so the whole allocation is burned")
	s.receiptRegLoya(moduleBase, s.moduleBalance(), "nothing is left in the module and nothing is missing")
}

// TestReporterWinKeepsVoterRewardClaimableFromReceivedFee is the AGAINST settlement of the previous
// test with a tipper who votes with the team, so a voter qualifies for the reward. The team and the
// tipper reach quorum, the round resolves at once and BeginBlocker executes it.
//
// The 5,000,000 allocation is half burned and half reserved for the tipper, 2,500,000 each. The
// winner must receive 195,000,000 and the module must be left holding exactly the reserve, so
// the tipper's claim succeeds and empties the module.
//
// A dispute that credited the 39,999,999 requested lets the payout succeed, but it overdraws
// the module by 79 loya, and the reserve it leaves is 79 short for the tipper's claim.
func (s *IntegrationTestSuite) TestReporterWinKeepsVoterRewardClaimableFromReceivedFee() {
	fx := s.newReportedStake()
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, fx.stake)
	tipper := sample.AccAddressBytes()
	s.seedTipper(tipper, fx.report)
	moduleBase := s.moduleBalance()

	d := s.receiptRegOpenVotingMajor(fx, bond, account)
	escrow := s.receiptRegEscrow(d)
	s.receiptRegFixtureLoya(fx.stake, escrow, "fixture: the reporter had the whole quote to escrow")
	funded := s.moduleBalance().Sub(moduleBase).Sub(escrow)
	s.receiptRegFixtureLoya(math.ZeroInt(), s.receiptRegStake(fx.reporter), "fixture: the reporter's whole stake is in escrow")

	executed, failure := s.receiptRegSettleAgainst(d, tipper)
	s.Require().True(executed, "the reporter-win settlement must execute from the coins the dispute holds: %s", failure)

	settled := s.receiptRegDispute(d.DisputeId)
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, d.DisputeId)
	s.Require().NoError(err)
	s.Equal(disputetypes.VoteResult_AGAINST, vote.VoteResult, "the team and the tipper reach quorum")
	s.True(settled.VoterReward.IsPositive(), "a voter qualifies, so part of the allocation is reserved")
	s.receiptRegLoya(escrow.Add(funded).Sub(settled.BurnAmount), s.receiptRegStake(fx.reporter), "the winner receives the escrow plus the funded fee less the allocation")
	s.receiptRegLoya(moduleBase.Add(settled.VoterReward), s.moduleBalance(), "the module is left with exactly the reserved voter reward")

	tipperBefore := s.receiptRegBalance(tipper)
	_, err = keeper.NewMsgServerImpl(s.Setup.Disputekeeper).ClaimReward(s.Setup.Ctx, &disputetypes.MsgClaimReward{CallerAddress: tipper.String(), DisputeId: d.DisputeId})
	s.Require().NoError(err, "the reserved voter reward must be there to claim")
	s.receiptRegLoya(settled.VoterReward, s.receiptRegBalance(tipper).Sub(tipperBefore), "the tipper is the only voter, so it receives the whole reserve")
	s.receiptRegLoya(moduleBase, s.moduleBalance(), "nothing is left in the module and nothing is missing")
}

// TestReporterWinLeavesFeesHeldForAnotherDisputeUntouched settles the AGAINST vote of the first
// reporter-win test in a module that also holds 3,000,000 of fees for another dispute. That
// dispute is on another reporter's report and is only partly funded, so its payer can withdraw
// the fee once the funding window has passed.
//
// The winner must receive exactly 195,000,000, which is the escrow plus the fee this dispute holds
// less the allocation. The module must then hold exactly the other dispute's 3,000,000 and its
// payer must get the fee back.
//
// A dispute that credited the 39,999,999 requested pays 195,000,000 out of the module anyway, the
// 79 loya it lacks come from the other dispute's fee, and that payer's withdrawal fails.
func (s *IntegrationTestSuite) TestReporterWinLeavesFeesHeldForAnotherDisputeUntouched() {
	fx := s.newReportedStake()
	moduleBase := s.moduleBalance()

	otherFee := math.NewInt(3_000_000)
	other, otherPayer := s.receiptRegOpenUnrelatedDispute(fx, otherFee)
	otherHeld := s.moduleBalance().Sub(moduleBase)
	s.receiptRegFixtureLoya(otherFee, otherHeld, "fixture: the other dispute's fee is held by the module")

	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, fx.stake)

	d := s.receiptRegOpenVotingMajor(fx, bond, account)
	escrow := s.receiptRegEscrow(d)
	s.receiptRegFixtureLoya(fx.stake, escrow, "fixture: the reporter had the whole quote to escrow")
	funded := s.moduleBalance().Sub(moduleBase).Sub(otherHeld).Sub(escrow)

	executed, failure := s.receiptRegSettleAgainst(d, nil)
	s.Require().True(executed, "the reporter-win settlement must execute: %s", failure)

	settled := s.receiptRegDispute(d.DisputeId)
	s.receiptRegLoya(escrow.Add(funded).Sub(settled.BurnAmount), s.receiptRegStake(fx.reporter), "the winner receives the escrow plus the funded fee less the allocation")
	s.receiptRegLoya(otherHeld, s.moduleBalance().Sub(moduleBase), "settling one dispute must leave the fees held for another untouched")

	// The funding window of the other dispute has passed, so it failed and its payer can withdraw.
	s.Equal(disputetypes.Failed, s.receiptRegDispute(other.DisputeId).DisputeStatus)
	_, err := keeper.NewMsgServerImpl(s.Setup.Disputekeeper).WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: otherPayer.String(),
		PayerAddress:  otherPayer.String(),
		Id:            other.DisputeId,
	})
	s.Require().NoError(err, "the other dispute's fee must still be there to withdraw")
	s.receiptRegLoya(otherFee, s.receiptRegBalance(otherPayer), "the other payer is refunded its whole fee")
	s.receiptRegLoya(moduleBase, s.moduleBalance(), "nothing is left in the module and nothing is missing")
}

// receiptRegOpenUnrelatedDispute opens a major dispute on another reporter's report and pays only
// fee of it, so the dispute stays in Prevote and the module holds the fee for its payer. The
// disputed reporter has no stake: escrow runs only when funding completes.
func (s *IntegrationTestSuite) receiptRegOpenUnrelatedDispute(fx reportedStake, fee math.Int) (disputetypes.Dispute, sdk.AccAddress) {
	reporter := sample.AccAddressBytes()
	report := fx.report
	report.Reporter = reporter.String()
	s.Require().NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter.Bytes(), report.MetaId), report))
	payer := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, fee)
	d := s.proposeDispute(report, disputetypes.Major, payer, fee, false)
	s.Require().Equal(disputetypes.Prevote, d.DisputeStatus)
	return d, payer
}

// receiptRegRequireLegacyReporterWin settles an AGAINST vote on an active dispute that was fully
// funded before receipts were credited, in the state receiptRegOpenLegacyVotingMajor builds. This
// is a legacy fixture, not current behavior. Its funding records show 100,000,000 of fee while
// the module holds 99,999,921 of fee coins, because the bonded payer's 79 loya loss was never
// recorded. The bonded payer's tracker and the module's coins are the real rounded ones.
//
// Settling it must not depend on coins that are not there. The module holds 100,000,000 of escrow
// and 99,999,921 of fee. Whatever allocation is burned or reserved, every coin must leave as the
// winner's stake, as burn, or as the reserve a voter can claim, and nothing may be missing.
//
// The vote is executed through the dispute BeginBlocker like any other AGAINST vote. Settling on
// the recorded fee asks the module for 200,000,000 in all, 79 loya more than it holds. Without a
// voter the payout and the burn overdraw the module and the execution fails. With a voter the
// payout succeeds and leaves a voter reserve that is 79 loya short of what the voter is owed.
func (s *IntegrationTestSuite) receiptRegRequireLegacyReporterWin(withVoter bool) {
	fx := s.newReportedStake()
	bond := s.receiptRegNewBondPayer(fx.vals[1])
	account := sample.AccAddressBytes()
	s.Setup.MintTokens(account, fx.stake)
	var tipper sdk.AccAddress
	if withVoter {
		tipper = sample.AccAddressBytes()
		s.seedTipper(tipper, fx.report)
	}
	moduleBase := s.moduleBalance()

	d := s.receiptRegOpenLegacyVotingMajor(fx, bond, account)
	escrow := s.receiptRegEscrow(d)
	s.receiptRegFixtureLoya(fx.stake, escrow, "fixture: the reporter had the whole quote to escrow")
	funded := s.moduleBalance().Sub(moduleBase).Sub(escrow)
	s.receiptRegFixtureLoya(fx.stake.SubRaw(79), funded, "fixture: 79 loya of the recorded fee never reached the module")
	s.Require().Equal(disputetypes.Voting, d.DisputeStatus, "fixture: the dispute is active")
	s.receiptRegFixtureLoya(fx.stake, d.SlashAmount, "fixture: the dispute records the full quote as slash")
	s.receiptRegFixtureLoya(fx.stake, d.DisputeFee, "fixture: the dispute records the full quote as fee")
	s.receiptRegFixtureLoya(fx.stake, d.FeeTotal, "fixture: the funding records show the full quote")
	s.receiptRegFixtureLoya(fx.stake.QuoRaw(20), d.BurnAmount, "fixture: the allocation is 5% of the recorded fee")
	bondInfo := s.payerInfo(d.DisputeId, bond.reporter)
	s.Require().True(bondInfo.FromBond, "fixture: the bonded payer is recorded as paying from bond")
	s.receiptRegFixtureLoya(math.NewInt(receiptRegBondOpening), bondInfo.Amount, "fixture: the bonded payer is recorded at the amount it requested")
	s.receiptRegFixtureLoya(math.NewInt(39_999_920), s.receiptRegTracked(d, bond.reporter).Total, "fixture: the tracker holds the coins really unbonded")
	s.receiptRegFixtureLoya(math.ZeroInt(), s.receiptRegStake(fx.reporter), "fixture: the reporter's whole stake is in escrow")
	supplyBefore := s.receiptRegSupply()

	executed, failure := s.receiptRegSettleAgainst(d, tipper)
	s.Require().True(executed, "a dispute funded before receipts were credited must still settle from the coins it holds: %s", failure)

	settled := s.receiptRegDispute(d.DisputeId)
	gain := s.receiptRegStake(fx.reporter)
	burned := supplyBefore.Sub(s.receiptRegSupply())
	s.True(gain.GTE(escrow), "the winner gets its escrow back")
	if !withVoter {
		s.receiptRegLoya(escrow.Add(funded), gain.Add(burned), "every coin the dispute holds is returned to the winner or burned")
		s.receiptRegLoya(moduleBase, s.moduleBalance(), "nothing is left in the module and nothing is missing")
		return
	}

	s.True(settled.VoterReward.IsPositive(), "a voter qualifies, so part of the allocation is reserved")
	s.receiptRegLoya(escrow.Add(funded), gain.Add(burned).Add(settled.VoterReward), "every coin the dispute holds is returned, burned or reserved")
	s.receiptRegLoya(moduleBase.Add(settled.VoterReward), s.moduleBalance(), "the module is left with exactly the reserved voter reward")
	tipperBefore := s.receiptRegBalance(tipper)
	_, err := keeper.NewMsgServerImpl(s.Setup.Disputekeeper).ClaimReward(s.Setup.Ctx, &disputetypes.MsgClaimReward{CallerAddress: tipper.String(), DisputeId: d.DisputeId})
	s.Require().NoError(err, "the reserved voter reward must be there to claim")
	s.receiptRegLoya(settled.VoterReward, s.receiptRegBalance(tipper).Sub(tipperBefore), "the tipper is the only voter, so it receives the whole reserve")
	s.receiptRegLoya(moduleBase, s.moduleBalance(), "nothing is left in the module and nothing is missing")
}

// TestLegacyNominalFundingReporterWinSettlesOnCoinsHeld settles an AGAINST vote on an active dispute
// funded before receipts were credited, when only the team votes and no voter qualifies. The whole
// allocation is burned and the module must end with nothing. See receiptRegRequireLegacyReporterWin.
func (s *IntegrationTestSuite) TestLegacyNominalFundingReporterWinSettlesOnCoinsHeld() {
	s.receiptRegRequireLegacyReporterWin(false)
}

// TestLegacyNominalFundingReporterWinKeepsVoterRewardClaimable settles the same legacy dispute with
// a tipper who votes with the team, so a voter qualifies. The module must be left holding exactly
// the reserved voter reward and the tipper's claim must succeed. See receiptRegRequireLegacyReporterWin.
func (s *IntegrationTestSuite) TestLegacyNominalFundingReporterWinKeepsVoterRewardClaimable() {
	s.receiptRegRequireLegacyReporterWin(true)
}
