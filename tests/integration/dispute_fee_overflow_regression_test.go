package integration_test

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	layertypes "github.com/tellor-io/layer/types"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// The cases below finish a partly collectible major dispute whose earlier payer
// drew its fee from bonded stake, so completing the funding returns part of that
// fee to the stake's owner. The owner's unbonding queue at the validator the
// stake came out of is already at the limit for user transactions. Funds,
// delegations, undelegations, jailing, the validator-set update and the unbonding
// payout all go through the real bank, staking, reporter and dispute keepers.
// The one direct state write is the reporter module's MaxValidatorPowerShare. It
// is the cap the refund must honor, and it is only set where a case needs a
// known amount of headroom (or none) on a validator. The reporter's stake beyond
// what the escrow can collect is removed with the existing leaveReporterSlashable
// helper, which also drops that reporter's own unbonding record so the escrow
// cannot reach it. That only sizes the shortfall. The refund path never touches it.
//
// The policy the cases expect: the refund is bonded only up to the headroom,
// with a single delegation. Whatever is left becomes an ordinary SDK unbonding
// entry of the original owner at the original validator, made by the keeper even
// though the queue is full, so it is paid when the unbonding period ends and
// never bonded past a cap or paid out at once.

// overflowCase is a major dispute opened by a from-bond payer with half the
// quote. The disputed reporter has 40% of the quoted slash left, so the escrow
// collects that much and 60% of each payer's fee is excess.
type overflowCase struct {
	disputeID    uint64
	hashID       []byte
	collected    math.Int
	accountShare math.Int
	bondKept     math.Int
	accountKept  math.Int
	bondExcess   math.Int
	bondPayer    sdk.AccAddress
	accountPayer sdk.AccAddress
	bondVal      sdk.ValAddress
	moduleStart  math.Int
}

// overflowState is what a case compares across the completing payment: the
// bond payer's liquid balance and bonded stake, its unbonding balance at the
// origin validator, the staking pools, the tokens recorded on bonded validators
// (which the staking module keeps equal to the bonded pool), the dispute module
// and the coin supply.
type overflowState struct {
	payerBank       math.Int
	payerBonded     math.Int
	originUnbonding math.Int
	bondedPool      math.Int
	notBondedPool   math.Int
	bondedSum       math.Int
	module          math.Int
	supply          math.Int
}

// overflowLedger is a validator's own totals next to one delegator's claim on it.
type overflowLedger struct {
	validator       stakingtypes.Validator
	delegatorShares math.LegacyDec
	delegatorTokens math.Int
}

// fundOverflowCase opens the dispute with the bond payer and then fills that
// payer's unbonding queue at the validator the bond came out of, with real
// undelegations. The second payer's transaction is left to each case so it can
// shape the validator set first.
func (s *IntegrationTestSuite) fundOverflowCase() overflowCase {
	fx := s.newReportedStake()
	s.leaveReporterSlashable(fx, math.NewInt(40_000_000))
	collected := s.delegationTokens(fx.reporter, fx.val)
	quote := fx.stake
	moduleStart := s.moduleBalance()

	bondPayer := sample.AccAddressBytes()
	bondVal := fx.vals[1]
	s.delegate(bondPayer, bondVal, math.NewInt(60_000_000))
	s.registerSelfReporter(bondPayer)
	accountPayer := sample.AccAddressBytes()
	s.Setup.MintTokens(accountPayer, quote)

	bondShare := quote.QuoRaw(2)
	accountShare := quote.Sub(bondShare)
	opened := s.proposeDispute(fx.report, disputetypes.Major, bondPayer, bondShare, true)
	s.Require().Equal(disputetypes.Prevote, opened.DisputeStatus)
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, opened.HashId, bondPayer)
	s.Require().NoError(err)
	s.Require().Len(tracked.TokenOrigins, 1)
	s.Require().True(sdk.ValAddress(tracked.TokenOrigins[0].ValidatorAddress).Equals(bondVal))
	s.Require().Equal(bondShare.String(), tracked.Total.String())

	maxEntries, err := s.Setup.Stakingkeeper.MaxEntries(s.Setup.Ctx)
	s.Require().NoError(err)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	height := s.Setup.Ctx.BlockHeight()
	for i := uint32(0); i < maxEntries; i++ {
		// Undelegations in one block merge into a single entry, so each one
		// gets its own height.
		height++
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(height)
		_, err = stakingMsgServer.Undelegate(s.Setup.Ctx, stakingtypes.NewMsgUndelegate(
			bondPayer.String(),
			bondVal.String(),
			sdk.NewCoin(s.Setup.Denom, math.NewInt(1_000)),
		))
		s.Require().NoError(err)
	}
	full, err := s.Setup.Stakingkeeper.HasMaxUnbondingDelegationEntries(s.Setup.Ctx, bondPayer, bondVal)
	s.Require().NoError(err)
	s.Require().True(full, "the bond payer's unbonding queue at the origin validator must be full")
	// The completing payment is a later block. An entry made at the same height
	// and completion time as the last one would merge into it instead of
	// going past the limit.
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(height + 1)

	bondKept := bondShare.Mul(collected).Quo(quote)
	return overflowCase{
		disputeID:    opened.DisputeId,
		hashID:       opened.HashId,
		collected:    collected,
		accountShare: accountShare,
		bondKept:     bondKept,
		accountKept:  accountShare.Mul(collected).Quo(quote),
		bondExcess:   bondShare.Sub(bondKept),
		bondPayer:    bondPayer,
		accountPayer: accountPayer,
		bondVal:      bondVal,
		moduleStart:  moduleStart,
	}
}

// completeOverflowFunding is the account payer's top-up. It reaches the quote,
// so the slash is escrowed and the bond payer's excess is returned in this call.
func (s *IntegrationTestSuite) completeOverflowFunding(c overflowCase) error {
	_, err := disputekeeper.NewMsgServerImpl(s.Setup.Disputekeeper).AddFeeToDispute(s.Setup.Ctx, &disputetypes.MsgAddFeeToDispute{
		Creator:     c.accountPayer.String(),
		DisputeId:   c.disputeID,
		Amount:      sdk.NewCoin(s.Setup.Denom, c.accountShare),
		PayFromBond: false,
	})
	return err
}

// jailOverflowValidators jails the validators and applies the staking
// end-block update, which takes them out of the bonded set for real.
func (s *IntegrationTestSuite) jailOverflowValidators(vals ...sdk.ValAddress) {
	for _, valAddr := range vals {
		validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
		s.Require().NoError(err)
		consAddr, err := validator.GetConsAddr()
		s.Require().NoError(err)
		s.Require().NoError(s.Setup.Stakingkeeper.Jail(s.Setup.Ctx, sdk.ConsAddress(consAddr)))
	}
	_, err := s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.Require().NoError(err)
	for _, valAddr := range vals {
		validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, valAddr)
		s.Require().NoError(err)
		s.Require().True(validator.Jailed && !validator.IsBonded(), "validator %s must be out of the bonded set", valAddr)
	}
}

func (s *IntegrationTestSuite) setOverflowPowerShare(share math.LegacyDec) {
	params, err := s.Setup.Reporterkeeper.Params.Get(s.Setup.Ctx)
	s.Require().NoError(err)
	params.MaxValidatorPowerShare = share
	s.Require().NoError(s.Setup.Reporterkeeper.Params.Set(s.Setup.Ctx, params))
}

// overflowCapRoom is the most a validator holding held tokens can take on while
// staying within share of the bonded total: the largest a with
// held+a <= share*(base+a).
func overflowCapRoom(held, base math.Int, share math.LegacyDec) math.Int {
	room := share.MulInt(base).Sub(held.ToLegacyDec())
	if !room.IsPositive() {
		return math.ZeroInt()
	}
	return room.Quo(math.LegacyOneDec().Sub(share)).TruncateInt()
}

func (s *IntegrationTestSuite) overflowStateOf(c overflowCase) overflowState {
	ctx := s.Setup.Ctx
	validators, err := s.Setup.Stakingkeeper.GetAllValidators(ctx)
	s.Require().NoError(err)
	byOperator := make(map[string]stakingtypes.Validator, len(validators))
	bondedSum := math.ZeroInt()
	for _, validator := range validators {
		byOperator[validator.OperatorAddress] = validator
		if validator.IsBonded() {
			bondedSum = bondedSum.Add(validator.Tokens)
		}
	}
	payerBonded := math.ZeroInt()
	s.Require().NoError(s.Setup.Stakingkeeper.IterateDelegatorDelegations(ctx, c.bondPayer, func(delegation stakingtypes.Delegation) bool {
		if validator := byOperator[delegation.ValidatorAddress]; validator.IsBonded() {
			payerBonded = payerBonded.Add(validator.TokensFromShares(delegation.Shares).TruncateInt())
		}
		return false
	}))
	ubd, err := s.Setup.Stakingkeeper.GetUnbondingDelegation(ctx, c.bondPayer, c.bondVal)
	s.Require().NoError(err)
	originUnbonding := math.ZeroInt()
	for _, entry := range ubd.Entries {
		originUnbonding = originUnbonding.Add(entry.Balance)
	}
	bondedPool, err := s.Setup.Stakingkeeper.TotalBondedTokens(ctx)
	s.Require().NoError(err)
	notBondedAddr := s.Setup.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName)
	return overflowState{
		payerBank:       s.Setup.Bankkeeper.GetBalance(ctx, c.bondPayer, s.Setup.Denom).Amount,
		payerBonded:     payerBonded,
		originUnbonding: originUnbonding,
		bondedPool:      bondedPool,
		notBondedPool:   s.Setup.Bankkeeper.GetBalance(ctx, notBondedAddr, s.Setup.Denom).Amount,
		bondedSum:       bondedSum,
		module:          s.moduleBalance(),
		supply:          s.Setup.Bankkeeper.GetSupply(ctx, s.Setup.Denom).Amount,
	}
}

func (s *IntegrationTestSuite) overflowLedgerOf(val sdk.ValAddress, del sdk.AccAddress) overflowLedger {
	validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, val)
	s.Require().NoError(err)
	ledger := overflowLedger{
		validator:       validator,
		delegatorShares: math.LegacyZeroDec(),
		delegatorTokens: math.ZeroInt(),
	}
	delegation, err := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, del, val)
	if err == nil {
		ledger.delegatorShares = delegation.Shares
		ledger.delegatorTokens = validator.TokensFromShares(delegation.Shares).TruncateInt()
	} else {
		s.Require().ErrorIs(err, stakingtypes.ErrNoDelegation)
	}
	return ledger
}

// overflowPowerIndex is the consensus power in each power-index entry that
// points at val. A validator's tokens decide which key its record belongs under.
func (s *IntegrationTestSuite) overflowPowerIndex(val sdk.ValAddress) []uint64 {
	iter, err := s.Setup.Stakingkeeper.ValidatorsPowerStoreIterator(s.Setup.Ctx)
	s.Require().NoError(err)
	defer iter.Close()
	var powers []uint64
	for ; iter.Valid(); iter.Next() {
		if bytes.Equal(iter.Value(), val) {
			// key: prefix (1 byte) | consensus power (8 bytes, big endian) | address length | inverted address
			powers = append(powers, binary.BigEndian.Uint64(iter.Key()[1:9]))
		}
	}
	return powers
}

// requireOverflowQueued checks what became of the part of the refund that was
// not bonded: an ordinary unbonding entry of the original owner at the original
// validator, made although that queue was already at the limit for user
// transactions. It must not be paid out at once, and the escrow and the
// retained fee stay with the dispute module. escrowFromNotBonded is the part of
// the escrow taken out of the not-bonded pool, which is where the disputed
// reporter's stake sits when its validator is not bonded.
func (s *IntegrationTestSuite) requireOverflowQueued(c overflowCase, before, after overflowState, queued, escrowFromNotBonded math.Int) {
	s.Equal(before.originUnbonding.Add(queued).String(), after.originUnbonding.String(),
		"the payer's unbonding balance at the origin validator must grow by the %s that was not bonded", queued)
	wantNotBonded := before.notBondedPool.Add(queued).Sub(escrowFromNotBonded)
	s.Equal(wantNotBonded.String(), after.notBondedPool.String(),
		"the not-bonded pool must receive the %s that was not bonded", queued)
	s.Equal(before.payerBank.String(), after.payerBank.String(), "the payer's liquid balance changed: the refund was paid out immediately")
	s.Equal(before.supply.String(), after.supply.String(), "completing the funding minted or burned coins")
	wantModule := c.moduleStart.Add(c.collected.MulRaw(2))
	s.Equal(wantModule.String(), after.module.String(), "the dispute module must hold exactly the escrow and the retained fee")
}

// requireOverflowPaidAtMaturity advances to the end of the ordinary unbonding
// period. The payer must be paid its old unbonding entries and the queued
// refund then, and not a second before.
func (s *IntegrationTestSuite) requireOverflowPaidAtMaturity(c overflowCase, before overflowState, queued math.Int) {
	payerBank := func() math.Int {
		return s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, c.bondPayer, s.Setup.Denom).Amount
	}
	start := payerBank()
	unbondingTime, err := s.Setup.Stakingkeeper.UnbondingTime(s.Setup.Ctx)
	s.Require().NoError(err)
	maturity := s.Setup.Ctx.BlockTime().Add(unbondingTime)

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(maturity.Add(-time.Second))
	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Equal(start.String(), payerBank().String(), "the payer was paid before the unbonding period ended")

	s.Setup.Ctx = s.Setup.Ctx.WithBlockTime(maturity.Add(time.Second))
	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.Require().NoError(err)
	wantPaid := start.Add(before.originUnbonding).Add(queued)
	s.Equal(wantPaid.String(), payerBank().String(), "at maturity the payer must receive its old unbonding entries and the %s queued by the refund", queued)
	s.True(s.unbondingBalance(c.bondPayer).IsZero(), "the payer must have no unbonding entries left after maturity")
}

// requireOverflowFunded checks the dispute reached voting with the collected
// slash as its fee, and that the escrow and each payer's retained share are intact.
func (s *IntegrationTestSuite) requireOverflowFunded(c overflowCase) {
	dispute, err := s.Setup.Disputekeeper.Disputes.Get(s.Setup.Ctx, c.disputeID)
	s.Require().NoError(err)
	s.Equal(disputetypes.Voting, dispute.DisputeStatus)
	s.Equal(c.collected.String(), dispute.SlashAmount.String(), "dispute slash amount")
	s.Equal(c.collected.String(), dispute.DisputeFee.String(), "dispute fee")
	s.Equal(c.collected.String(), dispute.FeeTotal.String(), "dispute fee total")
	s.Equal(c.collected.QuoRaw(20).String(), dispute.BurnAmount.String(), "dispute burn allocation")
	_, err = s.Setup.Disputekeeper.Votes.Get(s.Setup.Ctx, c.disputeID)
	s.NoError(err, "voting must have started")

	escrow, err := s.Setup.Reporterkeeper.DisputedDelegationAmounts.Get(s.Setup.Ctx, c.hashID)
	s.Require().NoError(err)
	s.Equal(c.collected.String(), escrow.Total.String(), "escrowed stake")
	s.Equal(c.bondKept.String(), s.payerInfo(c.disputeID, c.bondPayer).Amount.String(), "bond payer's retained fee")
	s.Equal(c.accountKept.String(), s.payerInfo(c.disputeID, c.accountPayer).Amount.String(), "account payer's retained fee")
	tracked, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, c.hashID, c.bondPayer)
	s.Require().NoError(err)
	s.Equal(c.bondKept.String(), tracked.Total.String(), "bond payer's stake still at risk")
}

// TestSplitRefundLeavesFallbackValidatorLedgerConsistent covers a split refund.
// The origin validator is out of the bonded set and its owner's queue is full, so
// the refund goes to the one bonded validator with room. That room is more than
// one consensus-power unit but less than the refund, so the refund has to split.
// Only the room may be bonded, with one delegation, and the remainder is queued
// for unbonding. A second delegation built from the validator record read before
// the first one writes totals that drop the first increase and leaves a stale
// power-index entry behind, while the delegation and the bonded pool keep both
// amounts.
func (s *IntegrationTestSuite) TestSplitRefundLeavesFallbackValidatorLedgerConsistent() {
	c := s.fundOverflowCase()

	s.jailOverflowValidators(c.bondVal)
	bondedVals, err := s.Setup.Stakingkeeper.GetBondedValidatorsByPower(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(bondedVals), 2)
	// Bonded validators are tried from the lowest power up, so the last one in
	// power order is the refund's fallback target.
	fallback := bondedVals[len(bondedVals)-1]
	fallbackAddr, err := sdk.ValAddressFromBech32(fallback.OperatorAddress)
	s.Require().NoError(err)

	// Size the cap so that validator has about five consensus-power units of
	// room and every other bonded validator has none. The escrow leaves the
	// bonded pool before the refund is returned, so the cap is sized on the
	// pool without it.
	bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	capBase := bondedBefore.Sub(c.collected)
	target := math.NewInt(5_000_000)
	share := math.LegacyNewDecFromInt(fallback.Tokens.Add(target)).Quo(math.LegacyNewDecFromInt(capBase.Add(target)))
	s.setOverflowPowerShare(share)
	room := overflowCapRoom(fallback.Tokens, capBase, share)
	s.Require().True(room.GT(layertypes.PowerReduction), "fallback room %s must exceed one consensus-power unit", room)
	s.Require().True(room.LT(c.bondExcess), "fallback room %s must be smaller than the refund %s", room, c.bondExcess)
	for _, other := range bondedVals[:len(bondedVals)-1] {
		s.Require().True(overflowCapRoom(other.Tokens, capBase, share).IsZero(), "validator %s must have no room", other.OperatorAddress)
	}

	before := s.overflowStateOf(c)
	ledgerBefore := s.overflowLedgerOf(fallbackAddr, c.bondPayer)
	s.Require().NoError(s.completeOverflowFunding(c), "the completing payment must succeed")
	after := s.overflowStateOf(c)
	ledgerAfter := s.overflowLedgerOf(fallbackAddr, c.bondPayer)

	// The stale write comes first: the validator's own totals must move by what
	// the payer's delegation moved by.
	validatorTokens := ledgerAfter.validator.Tokens.Sub(ledgerBefore.validator.Tokens)
	claimTokens := ledgerAfter.delegatorTokens.Sub(ledgerBefore.delegatorTokens)
	s.True(validatorTokens.Sub(claimTokens).Abs().LTE(math.OneInt()),
		"stale validator write: the fallback validator's tokens moved by %s but the payer's delegation moved by %s", validatorTokens, claimTokens)
	validatorShares := ledgerAfter.validator.DelegatorShares.Sub(ledgerBefore.validator.DelegatorShares)
	claimShares := ledgerAfter.delegatorShares.Sub(ledgerBefore.delegatorShares)
	s.Equal(claimShares.String(), validatorShares.String(),
		"stale validator write: the fallback validator's shares moved by %s but the payer's delegation shares moved by %s", validatorShares, claimShares)
	poolMoved := after.bondedPool.Sub(before.bondedPool)
	recordedMoved := after.bondedSum.Sub(before.bondedSum)
	s.Equal(recordedMoved.String(), poolMoved.String(),
		"stale validator write: the bonded pool moved by %s but the bonded validators' recorded tokens moved by %s", poolMoved, recordedMoved)
	wantPower := uint64(ledgerAfter.validator.ConsensusPower(s.Setup.Stakingkeeper.PowerReduction(s.Setup.Ctx)))
	s.Equal([]uint64{wantPower}, s.overflowPowerIndex(fallbackAddr),
		"stale validator write: the fallback validator must have one power-index entry, at the power of its current tokens")

	// Only the headroom is bonded, and the validator ends within its cap.
	bondedNow := after.payerBonded.Sub(before.payerBonded)
	s.Equal(bondedNow.String(), poolMoved.Add(c.collected).String(),
		"the bonded pool gained %s beyond the %s escrow that left it, but the payer's bonded stake moved by %s", poolMoved.Add(c.collected), c.collected, bondedNow)
	s.True(bondedNow.GT(layertypes.PowerReduction), "the headroom was not bonded: the payer's bonded stake moved by %s", bondedNow)
	s.True(bondedNow.LT(c.bondExcess), "the whole refund of %s was bonded past the headroom: the payer's bonded stake moved by %s", c.bondExcess, bondedNow)
	totalAfter, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	limit := share.MulInt(totalAfter).TruncateInt().AddRaw(1)
	s.True(ledgerAfter.validator.Tokens.LTE(limit), "the fallback validator holds %s, above its %s share of %s bonded tokens", ledgerAfter.validator.Tokens, share, totalAfter)

	// The rest of the refund is queued for unbonding.
	queued := c.bondExcess.Sub(bondedNow)
	s.requireOverflowQueued(c, before, after, queued, math.ZeroInt())
	s.requireOverflowFunded(c)
	s.requireOverflowPaidAtMaturity(c, before, queued)
}

// TestFullQueueRefundIsQueuedNotBondedPastCap covers a full queue with no
// headroom on any bonded validator. The completing payment must still succeed.
// The whole refund must come back as an unbonding entry at the original
// validator, not as bonded stake past the cap and not as liquid funds.
func (s *IntegrationTestSuite) TestFullQueueRefundIsQueuedNotBondedPastCap() {
	c := s.fundOverflowCase()

	// Every validator is far above a one-in-a-million share of the bonded pool.
	share := math.LegacyNewDecWithPrec(1, 6)
	s.setOverflowPowerShare(share)
	bondedBefore, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	capBase := bondedBefore.Sub(c.collected)
	bondedVals, err := s.Setup.Stakingkeeper.GetBondedValidatorsByPower(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(bondedVals)
	for _, validator := range bondedVals {
		s.Require().True(overflowCapRoom(validator.Tokens, capBase, share).IsZero(), "validator %s must have no room", validator.OperatorAddress)
	}

	before := s.overflowStateOf(c)
	s.Require().NoError(s.completeOverflowFunding(c), "a full unbonding queue must not stop the completing payment")
	after := s.overflowStateOf(c)

	s.True(after.payerBonded.LTE(before.payerBonded),
		"the payer's bonded stake rose from %s to %s: the refund was bonded past the cap while the queue was full", before.payerBonded, after.payerBonded)
	remaining := before.bondedPool.Sub(c.collected)
	s.True(after.bondedPool.LTE(remaining), "the bonded pool holds %s, more than the %s left once the escrow has left it", after.bondedPool, remaining)
	s.requireOverflowQueued(c, before, after, c.bondExcess, math.ZeroInt())
	s.requireOverflowFunded(c)
	s.requireOverflowPaidAtMaturity(c, before, c.bondExcess)
}

// TestFullQueueRefundWithoutBondedValidatorStillCompletesFunding covers the scan
// fallback in the refund path. With no bonded validator at all the headroom scan
// reports ErrExceedsMaxValidatorPowerShare, and the refund falls back to the
// origin validator's unbonding queue, which is full. The completing payment must
// still succeed instead of failing on the queue limit, and the refund is queued
// there for unbonding.
func (s *IntegrationTestSuite) TestFullQueueRefundWithoutBondedValidatorStillCompletesFunding() {
	c := s.fundOverflowCase()

	validators, err := s.Setup.Stakingkeeper.GetAllValidators(s.Setup.Ctx)
	s.Require().NoError(err)
	valAddrs := make([]sdk.ValAddress, 0, len(validators))
	for _, validator := range validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		s.Require().NoError(err)
		valAddrs = append(valAddrs, valAddr)
	}
	s.jailOverflowValidators(valAddrs...)
	bondedVals, err := s.Setup.Stakingkeeper.GetBondedValidatorsByPower(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Require().Empty(bondedVals, "no validator may be left bonded")

	before := s.overflowStateOf(c)
	s.Require().NoError(s.completeOverflowFunding(c), "a full unbonding queue must not stop the completing payment")
	after := s.overflowStateOf(c)

	s.True(after.payerBonded.LTE(before.payerBonded),
		"the payer's bonded stake rose from %s to %s with no bonded validator", before.payerBonded, after.payerBonded)
	s.True(after.bondedPool.LTE(before.bondedPool),
		"the bonded pool rose from %s to %s with no bonded validator", before.bondedPool, after.bondedPool)
	// The disputed reporter's validator is not bonded either, so the escrow came
	// out of the not-bonded pool.
	s.requireOverflowQueued(c, before, after, c.bondExcess, c.collected)
	s.requireOverflowFunded(c)
	s.requireOverflowPaidAtMaturity(c, before, c.bondExcess)
}
