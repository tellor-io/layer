package integration_test

import (
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) moduleBalance() math.Int {
	moduleAddr := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	return s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, moduleAddr, s.Setup.Denom).Amount
}

func (s *IntegrationTestSuite) delegationTokens(del sdk.AccAddress, val sdk.ValAddress) math.Int {
	ctx := s.Setup.Ctx
	delegation, err := s.Setup.Stakingkeeper.GetDelegation(ctx, del, val)
	s.NoError(err)
	validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, val)
	s.NoError(err)
	return validator.TokensFromShares(delegation.Shares).TruncateInt()
}

func (s *IntegrationTestSuite) delegate(del sdk.AccAddress, val sdk.ValAddress, amount math.Int) {
	s.Setup.MintTokens(del, amount)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, err := stakingMsgServer.Delegate(s.Setup.Ctx, stakingtypes.NewMsgDelegate(
		del.String(),
		val.String(),
		sdk.NewCoin(s.Setup.Denom, amount),
	))
	s.NoError(err)
}

func (s *IntegrationTestSuite) registerSelfReporter(addr sdk.AccAddress) {
	s.NoError(s.Setup.Reporterkeeper.Reporters.Set(s.Setup.Ctx, addr, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "reporter_moniker")))
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(s.Setup.Ctx, addr, reportertypes.NewSelection(addr, 1)))
}

type reportedStake struct {
	reporter sdk.AccAddress
	val      sdk.ValAddress
	vals     []sdk.ValAddress
	report   oracletypes.MicroReport
	stake    math.Int
}

func (s *IntegrationTestSuite) newReportedStake() reportedStake {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(time.Now()).WithBlockHeight(5)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{1000, 1000})
	reporter := sample.AccAddressBytes()
	stake := layertypes.PowerReduction.MulRaw(100)
	s.delegate(reporter, valAddrs[0], stake)
	s.Equal(stake, s.delegationTokens(reporter, valAddrs[0]))
	s.registerSelfReporter(reporter)

	qID, err := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	s.NoError(err)
	height := uint64(s.Setup.Ctx.BlockHeight())
	s.NoError(s.Setup.Reporterkeeper.ReportByBlock.Set(s.Setup.Ctx, collections.Join3(reporter.Bytes(), height, qID), reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{{
			DelegatorAddress: reporter.Bytes(),
			ValidatorAddress: valAddrs[0].Bytes(),
			Amount:           stake,
		}},
		Total: stake,
	}))
	report := oracletypes.MicroReport{
		Reporter:    reporter.String(),
		Power:       stake.Quo(layertypes.PowerReduction).Uint64(),
		QueryId:     qID,
		Value:       testValue,
		Timestamp:   s.Setup.Ctx.BlockTime().Add(-time.Hour),
		MetaId:      1,
		BlockNumber: height,
	}
	s.NoError(s.Setup.Oraclekeeper.Reports.Set(s.Setup.Ctx, collections.Join3(report.QueryId, reporter.Bytes(), report.MetaId), report))
	return reportedStake{reporter: reporter, val: valAddrs[0], vals: valAddrs, report: report, stake: stake}
}

func (s *IntegrationTestSuite) proposeDispute(report oracletypes.MicroReport, category disputetypes.DisputeCategory, payer sdk.AccAddress, fee math.Int, fromBond bool) disputetypes.Dispute {
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	disputeID := s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx)
	_, err := msgServer.ProposeDispute(s.Setup.Ctx, &disputetypes.MsgProposeDispute{
		Creator:          payer.String(),
		DisputedReporter: report.Reporter,
		ReportMetaId:     report.MetaId,
		ReportQueryId:    hex.EncodeToString(report.QueryId),
		DisputeCategory:  category,
		Fee:              sdk.NewCoin(s.Setup.Denom, fee),
		PayFromBond:      fromBond,
	})
	s.NoError(err)
	stored, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.NoError(err)
	return stored
}

func (s *IntegrationTestSuite) payerInfo(disputeID uint64, payer sdk.AccAddress) disputetypes.PayerInfo {
	info, err := s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(disputeID, payer.Bytes()))
	s.NoError(err)
	return info
}

func (s *IntegrationTestSuite) requireAdjustedEvent(disputeID uint64, quote, collected, burn math.Int) {
	disputeIDStr := strconv.FormatUint(disputeID, 10)
	found := false
	for _, ev := range s.Setup.Ctx.EventManager().Events() {
		if ev.Type != "dispute_fee_adjusted" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range ev.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["dispute_id"] != disputeIDStr {
			continue
		}
		found = true
		s.Equal(quote.String(), attrs["quote"])
		s.Equal(collected.String(), attrs["collected"])
		s.Equal(burn.String(), attrs["burn_amount"])
		s.NotContains(attrs, "payer")
		s.NotContains(attrs, "refund")
	}
	s.True(found, "missing dispute_fee_adjusted event")
}

func (s *IntegrationTestSuite) disputeEventAttr(eventType string, disputeID uint64, key string) (string, bool) {
	disputeIDStr := strconv.FormatUint(disputeID, 10)
	for _, ev := range s.Setup.Ctx.EventManager().Events() {
		if ev.Type != eventType {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range ev.Attributes {
			attrs[attr.Key] = attr.Value
		}
		if attrs["dispute_id"] != disputeIDStr {
			continue
		}
		value, ok := attrs[key]
		return value, ok
	}
	return "", false
}

func (s *IntegrationTestSuite) fundMinorThenMajor(fx reportedStake, payer sdk.AccAddress) (minorHeld, quote math.Int) {
	minorFee, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, fx.report, disputetypes.Minor)
	s.NoError(err)
	quote, err = s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, fx.report, disputetypes.Major)
	s.NoError(err)
	s.Equal(fx.stake, quote)

	minor := s.proposeDispute(fx.report, disputetypes.Minor, payer, minorFee, false)
	minorSnap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, minor.HashId)
	s.NoError(err)
	s.Equal(minorFee, minorSnap.Total)
	s.Equal(disputetypes.Voting, minor.DisputeStatus)
	return s.moduleBalance(), quote
}

func (s *IntegrationTestSuite) TestCollectedSlashResizesRoundOneFee() {
	fx := s.newReportedStake()
	payer := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, math.NewInt(300_000_000))
	balanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount
	minorHeld, quote := s.fundMinorThenMajor(fx, payer)

	major := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	collectedSnap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected := collectedSnap.Total
	s.True(collected.LT(quote))
	s.Equal(quote, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(quote, major.FeeTotal)
	burn := collected.QuoRaw(20)
	s.Equal(burn, major.BurnAmount)
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	info := s.payerInfo(major.DisputeId, payer)
	s.Equal(quote, info.Amount)
	s.False(info.FromBond)
	s.Equal(quote.Add(quote.QuoRaw(20)), balanceBefore.Sub(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount))
	s.Equal(minorHeld.Add(quote).Add(collected), s.moduleBalance())
	totalFee, ok := s.disputeEventAttr("new_dispute", major.DisputeId, "total_fee")
	s.True(ok)
	s.Equal(collected.String(), totalFee)
	feePaid, ok := s.disputeEventAttr("new_dispute", major.DisputeId, "fee_paid")
	s.True(ok)
	s.Equal(quote.String(), feePaid)
	s.requireAdjustedEvent(major.DisputeId, quote, collected, burn)
}

func (s *IntegrationTestSuite) fundSplitMajor() (fx reportedStake, major disputetypes.Dispute, quote, collected, minorHeld math.Int, accountPayer, bondPayer sdk.AccAddress, bondVal sdk.ValAddress) {
	fx = s.newReportedStake()
	accountPayer = sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	minorHeld, quote = s.fundMinorThenMajor(fx, accountPayer)

	bondVal = fx.vals[1]
	bondPayer = sample.AccAddressBytes()
	s.delegate(bondPayer, bondVal, math.NewInt(50_000_000))
	s.registerSelfReporter(bondPayer)
	accountShare := quote.MulRaw(60).QuoRaw(100)
	bondShare := quote.Sub(accountShare)
	partial := s.proposeDispute(fx.report, disputetypes.Major, accountPayer, accountShare, false)
	s.Equal(disputetypes.Prevote, partial.DisputeStatus)

	accountBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accountPayer, s.Setup.Denom).Amount
	bondBankBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount
	bondStakeBefore := s.delegationTokens(bondPayer, bondVal).Add(s.unbondingBalance(bondPayer))
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator: bondPayer.String(), DisputeId: partial.DisputeId,
		Amount: sdk.NewCoin(s.Setup.Denom, bondShare), PayFromBond: true,
	})
	s.NoError(err)

	major, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected = snap.Total
	s.True(collected.LT(quote))
	s.Equal(quote, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(quote, major.FeeTotal)
	s.Equal(disputetypes.Voting, major.DisputeStatus)
	s.Equal(minorHeld.Add(quote).Add(collected), s.moduleBalance())

	s.Equal(accountShare, s.payerInfo(major.DisputeId, accountPayer).Amount)
	s.False(s.payerInfo(major.DisputeId, accountPayer).FromBond)
	bondInfo := s.payerInfo(major.DisputeId, bondPayer)
	s.Equal(bondShare, bondInfo.Amount)
	s.True(bondInfo.FromBond)
	s.Equal(accountBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accountPayer, s.Setup.Denom).Amount)
	s.Equal(bondBankBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount)
	bondStakeAfter := s.delegationTokens(bondPayer, bondVal).Add(s.unbondingBalance(bondPayer))
	s.Equal(bondShare, bondStakeBefore.Sub(bondStakeAfter))

	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, major.HashId, bondPayer)
	s.NoError(err)
	s.Equal(bondShare, tracked.Total)
	originSum := math.ZeroInt()
	for _, origin := range tracked.TokenOrigins {
		originSum = originSum.Add(origin.Amount)
	}
	s.Equal(tracked.Total, originSum)
	feeAdded, ok := s.disputeEventAttr("fee_added_to_dispute", major.DisputeId, "amount")
	s.True(ok)
	s.Equal(bondShare.String(), feeAdded)
	s.requireAdjustedEvent(major.DisputeId, quote, collected, major.BurnAmount)
	return fx, major, quote, collected, minorHeld, accountPayer, bondPayer, bondVal
}

func (s *IntegrationTestSuite) TestTopUpResizeKeepsGrossPayerRows() {
	_, major, quote, collected, minorHeld, accountPayer, bondPayer, _ := s.fundSplitMajor()
	s.Equal(quote.MulRaw(60).QuoRaw(100), s.payerInfo(major.DisputeId, accountPayer).Amount)
	s.Equal(quote.MulRaw(40).QuoRaw(100), s.payerInfo(major.DisputeId, bondPayer).Amount)
	s.Equal(minorHeld.Add(quote).Add(collected), s.moduleBalance())
}

func (s *IntegrationTestSuite) TestFullSlashCollectionDoesNotAdjustFee() {
	fx := s.newReportedStake()
	payer := sample.AccAddressBytes()
	quote := fx.stake
	s.Setup.MintTokens(payer, quote)
	balanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount
	moduleBefore := s.moduleBalance()

	major := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	s.Equal(quote, major.SlashAmount)
	s.Equal(quote, major.DisputeFee)
	s.Equal(quote, major.FeeTotal)
	s.Equal(quote.QuoRaw(20), major.BurnAmount)
	s.Equal(quote, s.payerInfo(major.DisputeId, payer).Amount)
	s.Equal(quote, balanceBefore.Sub(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount))
	s.Equal(moduleBefore.Add(quote).Add(quote), s.moduleBalance())
	_, ok := s.disputeEventAttr("dispute_fee_adjusted", major.DisputeId, "quote")
	s.False(ok)
	totalFee, ok := s.disputeEventAttr("new_dispute", major.DisputeId, "total_fee")
	s.True(ok)
	s.Equal(quote.String(), totalFee)
}

func (s *IntegrationTestSuite) executeVote(dispute disputetypes.Dispute, result disputetypes.VoteResult) {
	vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, dispute.DisputeId)
	s.NoError(err)
	vote.VoteResult = result
	s.NoError(s.Setup.Disputekeeper.Votes.Set(s.Setup.Ctx, dispute.DisputeId, vote))
	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(dispute.DisputeEndTime.Add(time.Second))
	if _, err := s.Setup.Disputekeeper.Dust.Get(s.Setup.Ctx); err != nil {
		s.NoError(s.Setup.Disputekeeper.Dust.Set(s.Setup.Ctx, math.ZeroInt()))
	}
	s.NoError(s.Setup.Disputekeeper.ExecuteVote(s.Setup.Ctx, dispute.DisputeId))
}

func (s *IntegrationTestSuite) removeAllReporterStake(fx reportedStake) {
	s.leaveReporterSlashable(fx, math.OneInt())
	delegation, err := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, fx.reporter, fx.val)
	s.NoError(err)
	_, err = s.Setup.Stakingkeeper.Unbond(s.Setup.Ctx, fx.reporter, fx.val, delegation.Shares)
	s.NoError(err)
	_, err = s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, fx.reporter, fx.val)
	s.Error(err)
}

func feePaidFromStakePayerKey(hashId []byte, payer sdk.AccAddress) []byte {
	const prefix = "fee-paid-from-stake-payer\x00"
	key := make([]byte, 0, len(prefix)+4+len(hashId)+len(payer))
	key = append(key, prefix...)
	key = binary.BigEndian.AppendUint32(key, uint32(len(hashId)))
	key = append(key, hashId...)
	key = append(key, payer...)
	return key
}

func (s *IntegrationTestSuite) leaveReporterSlashable(fx reportedStake, left math.Int) {
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, err := stakingMsgServer.Undelegate(s.Setup.Ctx, stakingtypes.NewMsgUndelegate(
		fx.reporter.String(),
		fx.val.String(),
		sdk.NewCoin(s.Setup.Denom, fx.stake.Sub(left)),
	))
	s.NoError(err)
	ubd, err := s.Setup.Stakingkeeper.GetUnbondingDelegation(s.Setup.Ctx, fx.reporter, fx.val)
	s.NoError(err)
	s.NoError(s.Setup.Stakingkeeper.RemoveUnbondingDelegation(s.Setup.Ctx, ubd))
	s.True(s.delegationTokens(fx.reporter, fx.val).LTE(left))
	s.True(s.delegationTokens(fx.reporter, fx.val).IsPositive())
}

func (s *IntegrationTestSuite) TestCollectedSlashUnderTwentyKeepsQuoteAndChargesLaterRound() {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(15))
	payer := sample.AccAddressBytes()
	quote := fx.stake
	s.Setup.MintTokens(payer, quote.MulRaw(2))

	major := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	s.Equal(quote, major.SlashAmount)
	s.Equal(math.NewInt(15), major.DisputeFee)
	s.Equal(quote, major.FeeTotal)
	s.True(major.BurnAmount.IsZero())
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	major.DisputeStatus = disputetypes.Unresolved
	s.NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, major.DisputeId, major))
	round := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	s.Equal(uint64(2), round.DisputeRound)
	s.Equal(quote, round.SlashAmount)
	s.Equal(math.NewInt(15), round.DisputeFee)
	s.Equal(quote.Add(math.OneInt()), round.FeeTotal)
	s.Equal(math.OneInt(), round.BurnAmount)
}

func (s *IntegrationTestSuite) TestZeroCollectedSlashRejectsDispute() {
	fx := s.newReportedStake()
	s.removeAllReporterStake(fx)
	payer := sample.AccAddressBytes()
	quote := fx.stake
	s.Setup.MintTokens(payer, quote)
	balanceBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount
	moduleBefore := s.moduleBalance()
	disputeID := s.Setup.Disputekeeper.NextDisputeId(s.Setup.Ctx)

	cacheCtx, _ := s.Setup.Ctx.CacheContext()
	_, err := keeper.NewMsgServerImpl(s.Setup.Disputekeeper).ProposeDispute(cacheCtx, &disputetypes.MsgProposeDispute{
		Creator: payer.String(), DisputedReporter: fx.report.Reporter, ReportMetaId: fx.report.MetaId,
		ReportQueryId: hex.EncodeToString(fx.report.QueryId), DisputeCategory: disputetypes.Major,
		Fee: sdk.NewCoin(s.Setup.Denom, quote), PayFromBond: false,
	})
	s.ErrorIs(err, reportertypes.ErrReporterSlashShortfall)

	_, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeID)
	s.Error(err)
	_, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(disputeID, payer.Bytes()))
	s.Error(err)
	s.Equal(balanceBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount)
	s.Equal(moduleBefore, s.moduleBalance())
	reporter, err := s.Setup.Reporterkeeper.Reporters.Get(s.Setup.Ctx, fx.reporter)
	s.NoError(err)
	s.False(reporter.Jailed)
}

func (s *IntegrationTestSuite) TestFromBondPayerFundsTwiceThenResize() {
	fx := s.newReportedStake()
	accountPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	_, quote := s.fundMinorThenMajor(fx, accountPayer)

	bondPayer := sample.AccAddressBytes()
	s.delegate(bondPayer, fx.vals[1], math.NewInt(50_000_000))
	s.registerSelfReporter(bondPayer)
	first := math.NewInt(20_000_000)
	second := math.NewInt(10_000_000)
	partial := s.proposeDispute(fx.report, disputetypes.Major, bondPayer, first, true)
	s.Equal(disputetypes.Prevote, partial.DisputeStatus)
	firstTracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, partial.HashId, bondPayer)
	s.NoError(err)
	s.Equal(first, firstTracked.Total)
	s.Require().Len(firstTracked.TokenOrigins, 1)

	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator: bondPayer.String(), DisputeId: partial.DisputeId,
		Amount: sdk.NewCoin(s.Setup.Denom, second), PayFromBond: true,
	})
	s.NoError(err)
	secondTracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, partial.HashId, bondPayer)
	s.NoError(err)
	s.Equal(first.Add(second), secondTracked.Total)
	s.Require().Len(secondTracked.TokenOrigins, 1)

	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator: accountPayer.String(), DisputeId: partial.DisputeId,
		Amount: sdk.NewCoin(s.Setup.Denom, quote.Sub(first.Add(second))), PayFromBond: false,
	})
	s.NoError(err)

	major, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	s.True(snap.Total.LT(quote))
	s.Equal(quote, major.SlashAmount)
	s.Equal(snap.Total, major.DisputeFee)
	s.Equal(quote, major.FeeTotal)
	bondInfo := s.payerInfo(major.DisputeId, bondPayer)
	s.True(bondInfo.FromBond)
	s.Equal(first.Add(second), bondInfo.Amount)
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, major.HashId, bondPayer)
	s.NoError(err)
	s.Equal(bondInfo.Amount, tracked.Total)
	s.Equal(quote, bondInfo.Amount.Add(s.payerInfo(major.DisputeId, accountPayer).Amount))
}
