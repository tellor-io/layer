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

func (s *IntegrationTestSuite) requireAdjustedEvent(disputeID uint64, quote, collected, burn math.Int, refunds map[string]math.Int) {
	disputeIDStr := strconv.FormatUint(disputeID, 10)
	found := false
	for _, ev := range s.Setup.Ctx.EventManager().Events() {
		if ev.Type != "dispute_fee_adjusted" {
			continue
		}
		attrs := map[string]string{}
		var payers, refundVals []string
		for _, attr := range ev.Attributes {
			switch attr.Key {
			case "payer":
				payers = append(payers, attr.Value)
			case "refund":
				refundVals = append(refundVals, attr.Value)
			default:
				attrs[attr.Key] = attr.Value
			}
		}
		if attrs["dispute_id"] != disputeIDStr {
			continue
		}
		found = true
		s.Equal(quote.String(), attrs["quote"])
		s.Equal(collected.String(), attrs["collected"])
		s.Equal(burn.String(), attrs["burn_amount"])
		s.Equal(len(refunds), len(payers))
		s.Equal(len(payers), len(refundVals))
		got := make(map[string]string, len(payers))
		for i := range payers {
			got[payers[i]] = refundVals[i]
		}
		for addr, amt := range refunds {
			s.Equal(amt.String(), got[addr])
		}
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

func (s *IntegrationTestSuite) fundMinorThenMajor(fx reportedStake, payer sdk.AccAddress) (minorHeld math.Int, quote math.Int) {
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
	s.Equal(collected, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(collected, major.FeeTotal)
	burn := math.LegacyNewDecFromInt(collected).Quo(math.LegacyNewDec(20)).TruncateInt()
	s.Equal(burn, major.BurnAmount)
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	info := s.payerInfo(major.DisputeId, payer)
	s.Equal(collected, info.Amount)
	s.False(info.FromBond)
	s.Equal(collected, balanceBefore.Sub(s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, payer, s.Setup.Denom).Amount).Sub(quote.QuoRaw(20)))
	s.Equal(minorHeld.Add(collected).Add(collected), s.moduleBalance())
	s.True(collectedSnap.Total.IsPositive())

	totalFee, ok := s.disputeEventAttr("new_dispute", major.DisputeId, "total_fee")
	s.True(ok)
	s.Equal(collected.String(), totalFee)
	feePaid, ok := s.disputeEventAttr("new_dispute", major.DisputeId, "fee_paid")
	s.True(ok)
	s.Equal(quote.String(), feePaid)
	s.requireAdjustedEvent(major.DisputeId, quote, collected, burn, map[string]math.Int{
		payer.String(): quote.Sub(collected),
	})
}

func (s *IntegrationTestSuite) TestLaterRoundUsesResizedSlash() {
	fx := s.newReportedStake()
	payer := sample.AccAddressBytes()
	s.Setup.MintTokens(payer, math.NewInt(300_000_000))
	_, quote := s.fundMinorThenMajor(fx, payer)
	major := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)
	s.True(major.SlashAmount.LT(quote))

	major.DisputeStatus = disputetypes.Unresolved
	s.NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, major.DisputeId, major))
	round := s.proposeDispute(fx.report, disputetypes.Major, payer, quote, false)

	base := math.LegacyNewDecFromInt(major.SlashAmount).Quo(math.LegacyNewDec(20)).TruncateInt()
	roundFee := base.MulRaw(2)
	s.True(roundFee.LTE(major.SlashAmount))
	s.True(roundFee.LT(math.LegacyNewDecFromInt(quote).Quo(math.LegacyNewDec(20)).TruncateInt().MulRaw(2)))
	s.Equal(major.SlashAmount, round.SlashAmount)
	s.Equal(major.DisputeFee, round.DisputeFee)
	s.Equal(major.FeeTotal.Add(roundFee), round.FeeTotal)
	s.Equal(major.BurnAmount.Add(roundFee), round.BurnAmount)
	s.Equal(uint64(2), round.DisputeRound)
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

func (s *IntegrationTestSuite) fundSplitMajor() (fx reportedStake, major disputetypes.Dispute, quote, collected, minorHeld math.Int, accountPayer, bondPayer sdk.AccAddress, bondVal sdk.ValAddress) {
	fx = s.newReportedStake()
	accountPayer = sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	minorHeld, quote = s.fundMinorThenMajor(fx, accountPayer)

	bondVal = fx.vals[1]
	bondPayer = sample.AccAddressBytes()
	bondStake := math.NewInt(50_000_000)
	s.delegate(bondPayer, bondVal, bondStake)
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
		Creator:     bondPayer.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(s.Setup.Denom, bondShare),
		PayFromBond: true,
	})
	s.NoError(err)

	major, err = s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected = snap.Total
	s.True(collected.LT(quote))
	s.Equal(collected, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(collected, major.FeeTotal)
	s.Equal(disputetypes.Voting, major.DisputeStatus)
	s.Equal(minorHeld.Add(collected.MulRaw(2)), s.moduleBalance())

	accountKept := accountShare.Mul(collected).Quo(quote)
	bondKept := bondShare.Mul(collected).Quo(quote)
	s.Equal(collected, accountKept.Add(bondKept))
	s.Equal(accountKept, s.payerInfo(major.DisputeId, accountPayer).Amount)
	s.False(s.payerInfo(major.DisputeId, accountPayer).FromBond)
	bondInfo := s.payerInfo(major.DisputeId, bondPayer)
	s.Equal(bondKept, bondInfo.Amount)
	s.True(bondInfo.FromBond)

	s.Equal(accountBefore.Add(accountShare.Sub(accountKept)), s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, accountPayer, s.Setup.Denom).Amount)
	s.Equal(bondBankBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount)
	bondStakeAfter := s.delegationTokens(bondPayer, bondVal).Add(s.unbondingBalance(bondPayer))
	s.True(bondStakeBefore.Sub(bondStakeAfter).Sub(bondKept).Abs().LTE(math.OneInt()))

	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, major.HashId, bondPayer)
	s.NoError(err)
	s.Equal(bondInfo.Amount, tracked.Total)
	originSum := math.ZeroInt()
	for _, origin := range tracked.TokenOrigins {
		originSum = originSum.Add(origin.Amount)
	}
	s.Equal(tracked.Total, originSum)

	feeAdded, ok := s.disputeEventAttr("fee_added_to_dispute", major.DisputeId, "amount")
	s.True(ok)
	s.Equal(bondShare.String(), feeAdded)
	s.requireAdjustedEvent(major.DisputeId, quote, collected, major.BurnAmount, map[string]math.Int{
		accountPayer.String(): accountShare.Sub(accountKept),
		bondPayer.String():    bondShare.Sub(bondKept),
	})
	return fx, major, quote, collected, minorHeld, accountPayer, bondPayer, bondVal
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

func (s *IntegrationTestSuite) TestAdjustedFeeSupportClaimDrainsModule() {
	_, supportDispute, _, _, supportMinorHeld, supportAccount, supportBond, _ := s.fundSplitMajor()
	s.executeVote(supportDispute, disputetypes.VoteResult_SUPPORT)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := msgServer.WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: supportAccount.String(),
		PayerAddress:  supportAccount.String(),
		Id:            supportDispute.DisputeId,
	})
	s.NoError(err)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: supportBond.String(),
		PayerAddress:  supportBond.String(),
		Id:            supportDispute.DisputeId,
	})
	s.NoError(err)
	supportLeft := s.moduleBalance().Sub(supportMinorHeld)
	s.True(!supportLeft.IsNegative())
	s.True(supportLeft.LTE(math.NewInt(2)), "support settlement left %s above the earlier dispute", supportLeft)
}

func (s *IntegrationTestSuite) TestAdjustedFeeReporterWinDrainsModule() {
	_, reporterWin, _, _, winMinorHeld, _, _, _ := s.fundSplitMajor()
	s.executeVote(reporterWin, disputetypes.VoteResult_AGAINST)
	winLeft := s.moduleBalance().Sub(winMinorHeld)
	s.True(!winLeft.IsNegative())
	s.True(winLeft.LTE(math.NewInt(2)), "reporter-win settlement left %s above the earlier dispute", winLeft)
}

func (s *IntegrationTestSuite) TestFromBondExcessAtMaxUnbondingEntriesBondsPastCap() {
	fx := s.newReportedStake()
	accountPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	_, quote := s.fundMinorThenMajor(fx, accountPayer)

	bondVal := fx.vals[1]
	bondPayer := sample.AccAddressBytes()
	s.delegate(bondPayer, bondVal, math.NewInt(50_000_000))
	s.registerSelfReporter(bondPayer)

	params, err := s.Setup.Reporterkeeper.Params.Get(s.Setup.Ctx)
	s.NoError(err)
	params.MaxValidatorPowerShare = math.LegacyNewDecWithPrec(1, 6)
	s.NoError(s.Setup.Reporterkeeper.Params.Set(s.Setup.Ctx, params))

	maxEntries, err := s.Setup.Stakingkeeper.MaxEntries(s.Setup.Ctx)
	s.NoError(err)
	for i := uint32(0); i < maxEntries; i++ {
		_, err = s.Setup.Stakingkeeper.SetUnbondingDelegationEntry(
			s.Setup.Ctx,
			bondPayer,
			bondVal,
			int64(i+1),
			s.Setup.Ctx.BlockTime().Add(time.Duration(i+1)*time.Hour),
			math.NewInt(1),
		)
		s.NoError(err)
	}
	hasMax, err := s.Setup.Stakingkeeper.HasMaxUnbondingDelegationEntries(s.Setup.Ctx, bondPayer, bondVal)
	s.NoError(err)
	s.True(hasMax)

	accountShare := quote.MulRaw(60).QuoRaw(100)
	bondShare := quote.Sub(accountShare)
	partial := s.proposeDispute(fx.report, disputetypes.Major, accountPayer, accountShare, false)
	s.Equal(disputetypes.Prevote, partial.DisputeStatus)

	bankBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount
	delegationBefore := s.delegationTokens(bondPayer, bondVal)
	unbondingBefore := s.unbondingBalance(bondPayer)

	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator:     bondPayer.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(s.Setup.Denom, bondShare),
		PayFromBond: true,
	})
	s.NoError(err)

	major, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected := snap.Total
	s.True(collected.LT(quote))
	s.Equal(collected, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	bondKept := bondShare.Mul(collected).Quo(quote)
	s.Equal(bondKept, s.payerInfo(major.DisputeId, bondPayer).Amount)
	s.True(s.payerInfo(major.DisputeId, bondPayer).FromBond)
	s.Equal(bankBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount)
	s.Equal(unbondingBefore, s.unbondingBalance(bondPayer))
	ubd, err := s.Setup.Stakingkeeper.GetUnbondingDelegation(s.Setup.Ctx, bondPayer, bondVal)
	s.NoError(err)
	s.Equal(int(maxEntries), len(ubd.Entries))
	s.True(delegationBefore.Sub(s.delegationTokens(bondPayer, bondVal)).Sub(bondKept).Abs().LTE(math.OneInt()))
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

func (s *IntegrationTestSuite) stakeBalance(del sdk.AccAddress, val sdk.ValAddress) math.Int {
	delegated := math.ZeroInt()
	ctx := s.Setup.Ctx
	delegation, err := s.Setup.Stakingkeeper.GetDelegation(ctx, del, val)
	if err == nil {
		validator, valErr := s.Setup.Stakingkeeper.GetValidator(ctx, val)
		s.NoError(valErr)
		delegated = validator.TokensFromShares(delegation.Shares).TruncateInt()
	} else {
		s.ErrorIs(err, stakingtypes.ErrNoDelegation)
	}
	return delegated.Add(s.unbondingBalance(del))
}

func (s *IntegrationTestSuite) adjustFromBondTracker(hashId []byte, payer sdk.AccAddress, delta math.Int) reportertypes.DelegationsAmounts {
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, hashId, payer)
	s.NoError(err)
	s.NotEmpty(tracked.TokenOrigins)
	origin := tracked.TokenOrigins[0]
	updated := origin.Amount.Add(delta)
	s.True(updated.IsPositive())
	tracked.TokenOrigins[0] = &reportertypes.TokenOriginInfo{
		DelegatorAddress: origin.DelegatorAddress,
		ValidatorAddress: origin.ValidatorAddress,
		Amount:           updated,
	}
	tracked.Total = tracked.Total.Add(delta)
	s.NoError(s.Setup.Reporterkeeper.FeePaidFromStake.Set(s.Setup.Ctx, feePaidFromStakePayerKey(hashId, payer), tracked))
	got, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, hashId, payer)
	s.NoError(err)
	s.Equal(tracked.Total, got.Total)
	return got
}

func (s *IntegrationTestSuite) adjustDisputeModuleBalance(delta math.Int) {
	coins := sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, delta.Abs()))
	sink := sample.AccAddressBytes()
	if delta.IsNegative() {
		s.NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(s.Setup.Ctx, disputetypes.ModuleName, sink, coins))
		return
	}
	s.Setup.MintTokens(sink, delta)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(s.Setup.Ctx, sink, disputetypes.ModuleName, coins))
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

func (s *IntegrationTestSuite) TestFromBondTrackerShortfallStillRefunds() {
	fx := s.newReportedStake()
	accountPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	_, quote := s.fundMinorThenMajor(fx, accountPayer)

	bondVal := fx.vals[1]
	bondPayer := sample.AccAddressBytes()
	s.delegate(bondPayer, bondVal, math.NewInt(50_000_000))
	s.registerSelfReporter(bondPayer)

	bondShare := math.NewInt(20_000_000)
	partial := s.proposeDispute(fx.report, disputetypes.Major, bondPayer, bondShare, true)
	s.Equal(disputetypes.Prevote, partial.DisputeStatus)

	dust := math.NewInt(2)
	available := s.adjustFromBondTracker(partial.HashId, bondPayer, dust.Neg())
	s.adjustDisputeModuleBalance(dust.Neg())

	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator:     accountPayer.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(s.Setup.Denom, quote.Sub(bondShare)),
		PayFromBond: false,
	})
	s.NoError(err)

	major, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected := snap.Total
	s.True(collected.LT(quote))
	s.Equal(collected, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	bondInfo := s.payerInfo(major.DisputeId, bondPayer)
	s.True(bondInfo.FromBond)
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, major.HashId, bondPayer)
	s.NoError(err)
	s.Equal(bondInfo.Amount, tracked.Total)
	refund := available.Total.Sub(bondInfo.Amount)
	s.True(refund.IsPositive())
	accountInfo := s.payerInfo(major.DisputeId, accountPayer)
	s.requireAdjustedEvent(major.DisputeId, quote, collected, major.BurnAmount, map[string]math.Int{
		accountPayer.String(): quote.Sub(bondShare).Sub(accountInfo.Amount),
		bondPayer.String():    refund,
	})
	s.Equal(collected, bondInfo.Amount.Add(accountInfo.Amount))
}

func (s *IntegrationTestSuite) TestFromBondTrackerSurplusRoundsToZeroAndReturns() {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(5_000))
	accountPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, math.NewInt(300_000_000))
	quote, err := s.Setup.Disputekeeper.GetDisputeFee(s.Setup.Ctx, fx.report, disputetypes.Major)
	s.NoError(err)

	bondVal := fx.vals[1]
	bondPayer := sample.AccAddressBytes()
	s.delegate(bondPayer, bondVal, math.NewInt(50_000_000))
	s.registerSelfReporter(bondPayer)
	bankBefore := s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount

	opening := quote.Sub(math.NewInt(2_000))
	partial := s.proposeDispute(fx.report, disputetypes.Major, accountPayer, opening, false)
	s.Equal(disputetypes.Prevote, partial.DisputeStatus)

	bondShare := math.NewInt(1_000)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator:     bondPayer.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(s.Setup.Denom, bondShare),
		PayFromBond: true,
	})
	s.NoError(err)
	open, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	s.Equal(disputetypes.Prevote, open.DisputeStatus)

	stakeAfterPay := s.stakeBalance(bondPayer, bondVal)
	dust := math.NewInt(2)
	available := s.adjustFromBondTracker(partial.HashId, bondPayer, dust)
	s.adjustDisputeModuleBalance(dust)

	_, err = msgServer.AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator:     accountPayer.String(),
		DisputeId:   partial.DisputeId,
		Amount:      sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000)),
		PayFromBond: false,
	})
	s.NoError(err)

	major, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, partial.DisputeId)
	s.NoError(err)
	snap, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, major.HashId)
	s.NoError(err)
	collected := snap.Total
	s.True(collected.IsPositive())
	s.True(collected.LT(quote))
	s.Equal(collected, major.SlashAmount)
	s.Equal(collected, major.DisputeFee)
	s.Equal(disputetypes.Voting, major.DisputeStatus)

	_, err = s.Setup.Disputekeeper.DisputeFeePayer.Get(s.Setup.Ctx, collections.Join(major.DisputeId, bondPayer.Bytes()))
	s.Error(err)
	_, err = s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, major.HashId, bondPayer)
	s.Error(err)
	s.Equal(bankBefore, s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, bondPayer, s.Setup.Denom).Amount)
	s.True(s.stakeBalance(bondPayer, bondVal).Sub(stakeAfterPay.Add(available.Total)).Abs().LTE(math.OneInt()))
	s.Equal(collected, s.payerInfo(major.DisputeId, accountPayer).Amount)
}

func (s *IntegrationTestSuite) TestAdjustedFeeInvalidVoteDrainsModule() {
	fx, invalidDispute, _, collected, invalidMinorHeld, invalidAccount, invalidBond, _ := s.fundSplitMajor()
	reporterBefore := s.stakeBalance(fx.reporter, fx.val)
	s.executeVote(invalidDispute, disputetypes.VoteResult_INVALID)
	msgServer := keeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := msgServer.WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: invalidAccount.String(),
		PayerAddress:  invalidAccount.String(),
		Id:            invalidDispute.DisputeId,
	})
	s.NoError(err)
	_, err = msgServer.WithdrawFeeRefund(s.Setup.Ctx, &disputetypes.MsgWithdrawFeeRefund{
		CallerAddress: invalidBond.String(),
		PayerAddress:  invalidBond.String(),
		Id:            invalidDispute.DisputeId,
	})
	s.NoError(err)
	s.True(s.stakeBalance(fx.reporter, fx.val).Sub(reporterBefore.Add(collected)).Abs().LTE(math.OneInt()))
	invalidLeft := s.moduleBalance().Sub(invalidMinorHeld)
	s.True(!invalidLeft.IsNegative())
	s.True(invalidLeft.LTE(math.NewInt(2)), "invalid settlement left %s above the earlier dispute", invalidLeft)
}
