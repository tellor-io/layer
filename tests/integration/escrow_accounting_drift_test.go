// This file pins escrow accounting invariants: recorded amounts must match
// coins actually moved by Unbond, not theoretical share math.
package integration_test

import (
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	disputeabci "github.com/tellor-io/layer/x/dispute"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// escrowAgainstSkewedValidator escrows against a validator whose exchange rate
// is off 1.0 (tokens removed, shares unchanged).
func (s *IntegrationTestSuite) escrowAgainstSkewedValidator(powers []uint64, delegation, owed math.Int, removeTokens int64, hashId []byte) (sdk.AccAddress, sdk.ValAddress, math.Int, reportertypes.DelegationsAmounts) {
	ctx := s.Setup.Ctx
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, valAddrs, _ := s.createValidatorAccs(powers)
	val1 := valAddrs[0]

	selector := sample.AccAddressBytes()
	s.Setup.MintTokens(selector, delegation)
	_, err := stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, delegation),
	))
	s.NoError(err)

	val, err := s.Setup.Stakingkeeper.GetValidator(ctx, val1)
	s.NoError(err)
	_, err = s.Setup.Stakingkeeper.RemoveValidatorTokens(ctx, val, math.NewInt(removeTokens))
	s.NoError(err)

	reportHeight := uint64(ctx.BlockHeight())
	s.NoError(s.Setup.Reporterkeeper.Report.Set(ctx, collections.Join([]byte{}, collections.Join(selector.Bytes(), reportHeight)), reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{
			{
				DelegatorAddress: selector.Bytes(),
				ValidatorAddress: val1,
				Amount:           layertypes.PowerReduction,
			},
		},
		Total: layertypes.PowerReduction,
	}))

	disputeModule := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	before := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount
	s.True(before.IsZero(), "precondition: dispute module account must start empty so the shortfall is observable")

	s.NoError(s.Setup.Reporterkeeper.EscrowReporterStake(ctx, selector, 1, reportHeight, owed, []byte{}, hashId))

	received := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount.Sub(before)
	stored, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(ctx, hashId)
	s.NoError(err)
	return selector, val1, received, stored
}

func (s *IntegrationTestSuite) TestEscrowRecordedTotalExceedsCoinsReceivedFullRemoval() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	hashId := []byte("drift-full-removal")

	_, _, received, stored := s.escrowAgainstSkewedValidator(
		[]uint64{100}, math.NewInt(1_000_000), math.NewInt(1_000_000), 1, hashId)

	sum := math.ZeroInt()
	for _, origin := range stored.TokenOrigins {
		sum = sum.Add(origin.Amount)
	}
	s.Equal(stored.Total, sum, "origins must sum to Total")

	s.Equal(stored.Total, received,
		"DisputedDelegationAmounts.Total must equal the coins the dispute module actually received")
}

func (s *IntegrationTestSuite) TestEscrowRecordedTotalExceedsCoinsReceivedPartialDeduction() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	hashId := []byte("drift-partial-deduction")

	selector, val1, received, stored := s.escrowAgainstSkewedValidator(
		[]uint64{100}, math.NewInt(2_000_000), math.NewInt(1_000_000), 1, hashId)

	_, err := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, selector, val1)
	s.NoError(err, "partial deduction must leave the remainder of the delegation in place")

	s.Equal(stored.Total, received,
		"DisputedDelegationAmounts.Total must equal the coins the dispute module actually received")
}

func (s *IntegrationTestSuite) TestFeeFromReporterStakeOriginsSumExceedsTotal() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	ctx := s.Setup.Ctx
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	val1 := valAddrs[0]

	reporter := sample.AccAddressBytes()
	selector := sample.AccAddressBytes()
	delegation := math.NewInt(2_000_000)
	s.Setup.MintTokens(selector, delegation)
	_, err := stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, delegation),
	))
	s.NoError(err)
	s.NoError(s.Setup.Reporterkeeper.Selectors.Set(ctx, selector, reportertypes.NewSelection(reporter, 1)))

	val, err := s.Setup.Stakingkeeper.GetValidator(ctx, val1)
	s.NoError(err)
	_, err = s.Setup.Stakingkeeper.RemoveValidatorTokens(ctx, val, math.OneInt())
	s.NoError(err)

	disputeModule := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	before := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount

	fee := math.NewInt(1_000_000)
	hashId := []byte("drift-fee-tracker")
	s.NoError(s.Setup.Reporterkeeper.FeefromReporterStake(ctx, reporter, fee, hashId, true))

	received := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount.Sub(before)
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, hashId, reporter)
	s.NoError(err)

	s.Equal(tracked.Total, received, "fee tracker Total must equal coins the dispute module received")

	sum := math.ZeroInt()
	for _, origin := range tracked.TokenOrigins {
		sum = sum.Add(origin.Amount)
	}
	s.Equal(tracked.Total, sum,
		"FeePaidFromStake origins must sum to Total; FeeRefund distributes Amount*refund/Total per origin, so an inflated origins sum over-sends from the dispute module")
}

func (s *IntegrationTestSuite) TestInvalidVoteReturnSucceedsAgainstHonestSnapshot() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	hashId := []byte("drift-invalid-vote-return")

	// Extra validators keep val1 under the validator power cap during return.
	_, _, received, stored := s.escrowAgainstSkewedValidator(
		[]uint64{100, 300, 300}, math.NewInt(1_000_000), math.NewInt(1_000_000), 1, hashId)
	s.Equal(stored.Total, received, "precondition: snapshot must match the module's holdings")

	err := s.Setup.Disputekeeper.ReturnSlashedTokens(s.Setup.Ctx, disputetypes.Dispute{HashId: hashId}, math.ZeroInt())
	s.NoError(err,
		"returning slashed tokens for an invalid dispute must succeed against an honest snapshot")
}

func (s *IntegrationTestSuite) TestPendingExecutionCompletesAgainstHonestSnapshot() {
	blockTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1).WithBlockTime(blockTime)
	hashId := []byte("drift-pending-execution")

	selector, _, received, stored := s.escrowAgainstSkewedValidator(
		[]uint64{100, 300, 300}, math.NewInt(1_000_000), math.NewInt(1_000_000), 1, hashId)
	s.Equal(stored.Total, received, "precondition: snapshot must match the module's holdings")

	const disputeId = uint64(1)
	s.NoError(s.Setup.Disputekeeper.Disputes.Set(s.Setup.Ctx, disputeId, disputetypes.Dispute{
		HashId:           hashId,
		DisputeId:        disputeId,
		DisputeStatus:    disputetypes.Resolved,
		DisputeStartTime: blockTime.Add(-2 * time.Hour),
		DisputeEndTime:   blockTime.Add(-time.Hour),
		DisputeFee:       math.ZeroInt(),
		SlashAmount:      stored.Total,
		BurnAmount:       math.ZeroInt(),
		FeeTotal:         math.ZeroInt(),
		VoterReward:      math.ZeroInt(),
		InitialEvidence:  oracletypes.MicroReport{Reporter: selector.String(), BlockNumber: 1, Timestamp: blockTime.Add(-3 * time.Hour)},
		Open:             true,
		PendingExecution: true,
	}))
	s.NoError(s.Setup.Disputekeeper.Votes.Set(s.Setup.Ctx, disputeId, disputetypes.Vote{
		Id:         disputeId,
		VoteStart:  blockTime.Add(-2 * time.Hour),
		VoteEnd:    blockTime.Add(-time.Hour),
		VoteResult: disputetypes.VoteResult_INVALID,
		Executed:   false,
	}))

	for block := int64(2); block <= 4; block++ {
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(block).WithBlockTime(blockTime.Add(time.Duration(block) * time.Second))
		s.NoError(disputeabci.CheckClosedDisputesForExecution(s.Setup.Ctx, s.Setup.Disputekeeper))

		d, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, disputeId)
		s.NoError(err)
		s.False(d.PendingExecution, "block %d: dispute must eventually execute instead of retrying forever", block)
		vote, err := s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, disputeId)
		s.NoError(err)
		s.True(vote.Executed, "block %d: vote must eventually execute instead of retrying forever", block)
	}
}

func (s *IntegrationTestSuite) TestEscrowUnbondingExactEntryRemovalDoesNotLeaveZeroBalance() {
	blockTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1).WithBlockTime(blockTime)
	ctx := s.Setup.Ctx
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	val1 := valAddrs[0]

	selector := sample.AccAddressBytes()
	stake := math.NewInt(1000)
	firstEntry := math.NewInt(400)
	secondEntry := math.NewInt(600)
	s.Setup.MintTokens(selector, stake)
	_, err := stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, stake),
	))
	s.NoError(err)

	reportHeight := uint64(ctx.BlockHeight())
	hashId := []byte("ubd-exact-entry-cleanup")
	s.NoError(s.Setup.Reporterkeeper.Report.Set(ctx, collections.Join([]byte{}, collections.Join(selector.Bytes(), reportHeight)), reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{
			{
				DelegatorAddress: selector.Bytes(),
				ValidatorAddress: val1,
				Amount:           layertypes.PowerReduction,
			},
		},
		Total: layertypes.PowerReduction,
	}))

	_, err = stakingMsgServer.Undelegate(ctx, stakingtypes.NewMsgUndelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, firstEntry),
	))
	s.NoError(err)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(2).WithBlockTime(blockTime.Add(time.Second))
	ctx = s.Setup.Ctx
	_, err = stakingMsgServer.Undelegate(ctx, stakingtypes.NewMsgUndelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, secondEntry),
	))
	s.NoError(err)

	ubdBefore, err := s.Setup.Stakingkeeper.GetUnbondingDelegation(ctx, selector, val1)
	s.NoError(err)
	if s.Len(ubdBefore.Entries, 2, "precondition: the test needs two unbonding entries") {
		s.Equal(firstEntry, ubdBefore.Entries[0].Balance)
		s.Equal(secondEntry, ubdBefore.Entries[1].Balance)
	}

	disputeModule := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	before := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount
	s.NoError(s.Setup.Reporterkeeper.EscrowReporterStake(ctx, selector, 1, reportHeight, firstEntry, []byte{}, hashId))
	received := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount.Sub(before)
	s.Equal(firstEntry, received)

	stored, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(ctx, hashId)
	s.NoError(err)
	s.Equal(firstEntry, stored.Total)

	ubdAfter, err := s.Setup.Stakingkeeper.GetUnbondingDelegation(ctx, selector, val1)
	s.NoError(err)
	remaining := math.ZeroInt()
	for _, entry := range ubdAfter.Entries {
		s.True(entry.Balance.IsPositive(), "spent unbonding entries must be removed, not retained with zero balance")
		remaining = remaining.Add(entry.Balance)
	}
	s.Equal(secondEntry, remaining)
}

func (s *IntegrationTestSuite) TestReturnFeeToStakeWithIntendedAmountExceedsActualBondFunding() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	ctx := s.Setup.Ctx
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100, 300, 300})
	val1 := valAddrs[0]
	payer := sample.AccAddressBytes()
	actualFromBond := math.NewInt(999)
	intendedFee := math.NewInt(1000)
	hashId := []byte("fee-intended-exceeds-actual")

	s.Setup.MintTokens(payer, actualFromBond)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, payer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, actualFromBond))))
	// Tracker records actual bond-funded amount; payer may still claim intended fee.
	s.NoError(s.Setup.Reporterkeeper.FeePaidFromStake.Set(ctx, hashId, reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{
			{
				DelegatorAddress: payer.Bytes(),
				ValidatorAddress: val1,
				Amount:           actualFromBond,
			},
		},
		Total: actualFromBond,
	}))

	before := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), s.Setup.Denom).Amount
	err := s.Setup.Disputekeeper.ReturnFeetoStake(ctx, hashId, payer, intendedFee)
	if s.NoError(err, "from-bond fee refunds must not ask the dispute module to return more than the stake fee tracker actually collected") {
		after := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), s.Setup.Denom).Amount
		s.Equal(actualFromBond, before.Sub(after))
	}
}

func (s *IntegrationTestSuite) TestRefundDisputeFeeFromBondUsesActualFundingAfterBurn() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	ctx := s.Setup.Ctx
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100, 300, 300})
	val1 := valAddrs[0]
	payer := sample.AccAddressBytes()
	actualFromBond := math.NewInt(999)
	intendedFee := math.NewInt(1000)
	intendedRefund, _ := disputekeeper.CalculateRefundAmount(intendedFee, intendedFee)
	expectedRefund := disputekeeper.CalculateActualStakeFeeRefundAmount(actualFromBond, intendedFee, intendedRefund)
	hashId := []byte("fee-actual-after-burn")

	s.Setup.MintTokens(payer, actualFromBond)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, payer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, actualFromBond))))
	s.NoError(s.Setup.Reporterkeeper.FeePaidFromStake.Set(ctx, hashId, reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{
			{
				DelegatorAddress: payer.Bytes(),
				ValidatorAddress: val1,
				Amount:           actualFromBond,
			},
		},
		Total: actualFromBond,
	}))

	feeBurn := intendedFee.Sub(intendedRefund)
	s.NoError(s.Setup.Bankkeeper.BurnCoins(ctx, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, feeBurn))))
	before := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), s.Setup.Denom).Amount
	s.Equal(expectedRefund, before)

	remainder, err := s.Setup.Disputekeeper.RefundDisputeFee(ctx, payer, disputetypes.PayerInfo{
		Amount:   intendedFee,
		FromBond: true,
	}, intendedFee, hashId)
	s.NoError(err, "from-bond fee refund after burn must not exceed actual fee funding left in the dispute module")
	s.True(remainder.IsZero())
	after := s.Setup.Bankkeeper.GetBalance(ctx, s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName), s.Setup.Denom).Amount
	s.True(after.IsZero())
}

func (s *IntegrationTestSuite) TestSupportFeeRefundRewardsActualEscrowedReporterBond() {
	blockTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1).WithBlockTime(blockTime)
	ctx := s.Setup.Ctx
	hashId := []byte("support-actual-reporter-bond")
	intendedSlash := math.NewInt(1_000_000)

	_, _, received, stored := s.escrowAgainstSkewedValidator(
		[]uint64{100, 300, 300}, math.NewInt(1_000_000), intendedSlash, 1, hashId)
	s.Equal(stored.Total, received)
	s.Equal(math.NewInt(999_999), received, "precondition: skewed validator must create a one-loya measured shortfall")

	feePayer := sample.AccAddressBytes()
	s.Setup.MintTokens(feePayer, intendedSlash)
	s.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(ctx, feePayer, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, intendedSlash))))
	feeBurn := intendedSlash.QuoRaw(20)
	s.NoError(s.Setup.Bankkeeper.BurnCoins(ctx, disputetypes.ModuleName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, feeBurn))))

	const disputeId = uint64(1)
	s.NoError(s.Setup.Disputekeeper.Disputes.Set(ctx, disputeId, disputetypes.Dispute{
		HashId:           hashId,
		DisputeId:        disputeId,
		DisputeStatus:    disputetypes.Resolved,
		DisputeStartTime: blockTime.Add(-2 * time.Hour),
		DisputeEndTime:   blockTime.Add(-time.Hour),
		SlashAmount:      intendedSlash,
		DisputeFee:       intendedSlash,
		FeeTotal:         intendedSlash,
		Open:             true,
		PendingExecution: false,
	}))
	s.NoError(s.Setup.Disputekeeper.Votes.Set(ctx, disputeId, disputetypes.Vote{
		Id:         disputeId,
		VoteStart:  blockTime.Add(-2 * time.Hour),
		VoteEnd:    blockTime.Add(-time.Hour),
		VoteResult: disputetypes.VoteResult_SUPPORT,
		Executed:   true,
	}))
	s.NoError(s.Setup.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(disputeId, feePayer.Bytes()), disputetypes.PayerInfo{
		Amount:   intendedSlash,
		FromBond: false,
	}))

	disputeModule := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	before := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount
	s.Equal(received.Add(intendedSlash).Sub(feeBurn), before)

	msgServer := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper)
	_, err := msgServer.WithdrawFeeRefund(ctx, &disputetypes.MsgWithdrawFeeRefund{
		Id:            disputeId,
		PayerAddress:  feePayer.String(),
		CallerAddress: feePayer.String(),
	})
	s.NoError(err, "support refunds must reward only the actual reporter bond collected")
	after := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount
	s.True(after.IsZero(), "refund plus reporter-bond reward should exactly drain the module balance")
}

func (s *IntegrationTestSuite) TestEscrowReporterStakeRedelegationFanOutCollectsAllDestinations() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	ctx := s.Setup.Ctx
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100, 100, 100})
	val1, val2, val3 := valAddrs[0], valAddrs[1], valAddrs[2]

	selector := sample.AccAddressBytes()
	firstDst := math.NewInt(400)
	secondDst := math.NewInt(600)
	total := firstDst.Add(secondDst)
	s.Setup.MintTokens(selector, total)
	_, err := stakingMsgServer.Delegate(ctx, stakingtypes.NewMsgDelegate(
		selector.String(),
		val1.String(),
		sdk.NewCoin(s.Setup.Denom, total),
	))
	s.NoError(err)

	reportHeight := uint64(ctx.BlockHeight())
	hashId := []byte("redelegation-fan-out")
	s.NoError(s.Setup.Reporterkeeper.Report.Set(ctx, collections.Join([]byte{}, collections.Join(selector.Bytes(), reportHeight)), reportertypes.DelegationsAmounts{
		TokenOrigins: []*reportertypes.TokenOriginInfo{
			{
				DelegatorAddress: selector.Bytes(),
				ValidatorAddress: val1,
				Amount:           layertypes.PowerReduction,
			},
		},
		Total: layertypes.PowerReduction,
	}))

	_, err = stakingMsgServer.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(
		selector.String(),
		val1.String(),
		val2.String(),
		sdk.NewCoin(s.Setup.Denom, firstDst),
	))
	s.NoError(err)
	_, err = stakingMsgServer.BeginRedelegate(ctx, stakingtypes.NewMsgBeginRedelegate(
		selector.String(),
		val1.String(),
		val3.String(),
		sdk.NewCoin(s.Setup.Denom, secondDst),
	))
	s.NoError(err)

	disputeModule := s.Setup.Accountkeeper.GetModuleAddress(disputetypes.ModuleName)
	before := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount
	s.NoError(s.Setup.Reporterkeeper.EscrowReporterStake(ctx, selector, 1, reportHeight, total, []byte{}, hashId))
	received := s.Setup.Bankkeeper.GetBalance(ctx, disputeModule, s.Setup.Denom).Amount.Sub(before)

	stored, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(ctx, hashId)
	s.NoError(err)
	s.Equal(total, received, "escrow should collect stake across every redelegation destination from the source validator")
	s.Equal(total, stored.Total)
	sum := math.ZeroInt()
	for _, origin := range stored.TokenOrigins {
		sum = sum.Add(origin.Amount)
	}
	s.Equal(stored.Total, sum)
}
