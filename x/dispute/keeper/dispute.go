package keeper

import (
	"context"
	"crypto/sha256"
	"fmt"
	gomath "math"
	"math/big"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

// Get dispute by reporter key
func (k Keeper) GetDisputeByReporter(ctx sdk.Context, r oracletypes.MicroReport, c types.DisputeCategory) (types.Dispute, error) {
	key := []byte(k.ReporterKey(ctx, r, c))

	id, err := k.Disputes.Indexes.DisputeByReporter.MatchExact(ctx, key)
	if err != nil {
		return types.Dispute{}, err
	}

	return k.Disputes.Get(ctx, id)
}

// Get next dispute id
func (k Keeper) NextDisputeId(ctx sdk.Context) uint64 {
	rng := new(collections.Range[uint64]).EndExclusive(gomath.MaxUint64).Descending()
	// start dispute count at 1
	id := uint64(1)
	err := k.Disputes.Walk(ctx, rng, func(k uint64, _ types.Dispute) (stop bool, err error) {
		id = k + 1
		return true, nil
	})
	if err != nil {
		return 0
	}
	return id
}

// Generate hash id
func (k Keeper) HashId(ctx sdk.Context, r oracletypes.MicroReport, c types.DisputeCategory) [32]byte {
	params := types.DisputeParams{
		Report:   &r,
		Category: c,
	}
	return sha256.Sum256(k.cdc.MustMarshal(&params))
}

// Make a reporter key by combining reporter address and a hash of dispute params
func (k Keeper) ReporterKey(ctx sdk.Context, r oracletypes.MicroReport, c types.DisputeCategory) string {
	return fmt.Sprintf("%s:%x", r.Reporter, k.HashId(ctx, r, c))
}

// Set new dispute
func (k Keeper) SetNewDispute(ctx sdk.Context, msg types.MsgProposeDispute) error {
	disputeId := k.NextDisputeId(ctx)
	hashId := k.HashId(ctx, *msg.Report, msg.DisputeCategory)
	// slash amount
	disputeFee, err := k.GetDisputeFee(ctx, msg.Report.Reporter, msg.DisputeCategory)
	if err != nil {
		return err
	}

	feeList := make([]types.PayerInfo, 0)

	if msg.Fee.Amount.GT(disputeFee) {
		msg.Fee.Amount = disputeFee
	}
	fivePercent := disputeFee.MulRaw(1).QuoRaw(20)
	dispute := types.Dispute{
		HashId:            hashId[:],
		DisputeId:         disputeId,
		DisputeCategory:   msg.DisputeCategory,
		DisputeStatus:     types.Prevote,
		DisputeStartTime:  ctx.BlockTime(),
		DisputeEndTime:    ctx.BlockTime().Add(ONE_DAY), // one day to fully pay fee
		DisputeStartBlock: ctx.BlockHeight(),
		DisputeRound:      1,
		SlashAmount:       disputeFee,
		// burn amount is calculated as 5% of dispute fee
		BurnAmount:     fivePercent,
		DisputeFee:     disputeFee.Sub(fivePercent),
		ReportEvidence: *msg.Report,
		FeePayers: append(feeList, types.PayerInfo{
			PayerAddress: msg.Creator,
			Amount:       msg.Fee.Amount,
			FromBond:     msg.PayFromBond,
			BlockNumber:  ctx.BlockHeight(),
		}),
		FeeTotal:       msg.Fee.Amount,
		PrevDisputeIds: []uint64{disputeId},
	}
	// Pay the dispute fee
	if err := k.PayDisputeFee(ctx, msg.Creator, msg.Fee, msg.PayFromBond); err != nil {
		return err
	}
	// if the paid fee is equal to the slash amount, then slash validator and jail
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		if err := k.SlashAndJailReporter(ctx, dispute.ReportEvidence, dispute.DisputeCategory); err != nil {
			return err
		}
		// extend dispute end time by 3 days, 2 for voting and 1 to allow for more rounds
		dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
		dispute.DisputeStatus = types.Voting
		if err := k.SetStartVote(ctx, dispute.DisputeId); err != nil { // starting voting immediately
			return err
		}
	}

	return k.SetDispute(ctx, dispute)
}

// Slash and jail reporter
func (k Keeper) SlashAndJailReporter(ctx sdk.Context, report oracletypes.MicroReport, category types.DisputeCategory) error {
	reporterAddr := sdk.MustAccAddressFromBech32(report.Reporter)

	slashFactor, jailDuration, err := k.GetSlashPercentageAndJailDuration(category)
	if err != nil {
		return err
	}
	amount := math.NewInt(report.Power).Mul(layertypes.PowerReduction)
	slashAmount := math.LegacyNewDecFromInt(amount).Mul(slashFactor)
	err = k.reporterKeeper.EscrowReporterStake(ctx, reporterAddr, report.Power, report.BlockNumber, slashAmount.TruncateInt())
	if err != nil {
		return err
	}
	return k.jailReporter(ctx, reporterAddr, jailDuration)
}

func (k Keeper) jailReporter(ctx context.Context, repAddr sdk.AccAddress, jailDuration int64) error {
	// noop for major duration, reporter is removed from store so no need to jail
	if jailDuration == gomath.MaxInt64 {
		return nil
	}
	return k.reporterKeeper.JailReporter(ctx, repAddr, jailDuration)
}

// Get percentage of slash amount based on category
func (k Keeper) GetSlashPercentageAndJailDuration(category types.DisputeCategory) (math.LegacyDec, int64, error) {
	switch category {
	case types.Warning:
		return math.LegacyNewDecWithPrec(1, 2), 0, nil // 1%
	case types.Minor:
		return math.LegacyNewDecWithPrec(5, 2), 600, nil // 5%
	case types.Major:
		return math.LegacyNewDecWithPrec(1, 0), gomath.MaxInt64, nil // 100%
	default:
		return math.LegacyDec{}, 0, types.ErrInvalidDisputeCategory
	}
}

// Get dispute fee
func (k Keeper) GetDisputeFee(ctx sdk.Context, repAddr string, category types.DisputeCategory) (math.Int, error) {
	reporterAddr := sdk.MustAccAddressFromBech32(repAddr)
	reporter, err := k.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		return math.Int{}, err
	}

	stake := reporter.TotalTokens
	switch category {
	case types.Warning:
		// calculate 1 percent of bond
		return stake.MulRaw(1).QuoRaw(100), nil
	case types.Minor:
		// calculate 5 percent of bond
		return stake.MulRaw(5).QuoRaw(100), nil
	case types.Major:
		// calculate 100 percent of bond
		return stake, nil
	default:
		return math.Int{}, types.ErrInvalidDisputeCategory
	}
}

// Update existing dispute when conditions are met
func (k Keeper) AddDisputeRound(ctx sdk.Context, dispute types.Dispute, msg types.MsgProposeDispute) error {
	if dispute.DisputeStatus != types.Unresolved {
		return fmt.Errorf("can't start a new round for this dispute %d; dispute status %s", dispute.DisputeId, dispute.DisputeStatus)
	}
	// if dispute is not unresovled and dispute end time is before current block time then we can't update it
	if dispute.DisputeEndTime.Before(ctx.BlockTime()) {
		return fmt.Errorf("this dispute is expired, can't start new round %d", dispute.DisputeId)
	}

	fee := func(fivePercent math.Int, round int64) math.Int {
		base := new(big.Int).Exp(big.NewInt(2), big.NewInt(round), nil)
		return fivePercent.Mul(math.NewIntFromBigInt(base))
	}
	fivePercent := dispute.SlashAmount.MulRaw(1).QuoRaw(20)
	roundFee := fee(fivePercent, int64(dispute.DisputeRound))
	if roundFee.GT(dispute.SlashAmount) {
		roundFee = dispute.SlashAmount
	}

	if msg.Fee.Amount.LT(roundFee) {
		return fmt.Errorf("fee amount is less than amount required")
	} else {
		msg.Fee.Amount = roundFee
	}

	// Pay the dispute fee
	if err := k.PayDisputeFee(ctx, msg.Creator, msg.Fee, msg.PayFromBond); err != nil {
		return err
	}

	dispute.BurnAmount = dispute.BurnAmount.Add(roundFee)
	dispute.FeeTotal = dispute.FeeTotal.Add(msg.Fee.Amount)
	disputeId := k.NextDisputeId(ctx)
	dispute.DisputeId = disputeId
	dispute.DisputeStatus = types.Voting // starting voting immediately
	dispute.DisputeStartTime = ctx.BlockTime()
	// add 3 days to block time
	dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
	dispute.DisputeStartBlock = ctx.BlockHeight()
	dispute.DisputeRound++
	// from previous dispute id from open disputes
	err := k.removeId(ctx, dispute.PrevDisputeIds[len(dispute.PrevDisputeIds)-1])
	if err != nil {
		return err
	}
	dispute.PrevDisputeIds = append(dispute.PrevDisputeIds, disputeId)

	err = k.SetDispute(ctx, dispute)
	if err != nil {
		return err
	}
	err = k.SetStartVote(ctx, dispute.DisputeId) // starting voting immediately
	if err != nil {
		return err
	}

	return nil
}

// remove dispute id from opendisputes after adding a new round
func (k Keeper) removeId(ctx sdk.Context, disputeId uint64) error {
	openDisputes, err := k.OpenDisputes.Get(ctx)
	if err != nil {
		return err
	}
	for i, id := range openDisputes.Ids {
		if id == disputeId {
			openDisputes.Ids[i] = openDisputes.Ids[len(openDisputes.Ids)-1]
			openDisputes.Ids = openDisputes.Ids[:len(openDisputes.Ids)-1]
			err = k.OpenDisputes.Set(ctx, openDisputes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Add time to dispute end time
func (k Keeper) AddTimeToDisputeEndTime(ctx sdk.Context, id uint64, timeToAdd time.Duration) error {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}
	dispute.DisputeEndTime = dispute.DisputeEndTime.Add(timeToAdd)
	return k.Disputes.Set(ctx, dispute.DisputeId, dispute)
}

// Append dispute id to open dispute ids
func (k Keeper) AppendDisputeIdToOpenDisputeIds(ctx sdk.Context, disputeId uint64) error {
	openDisputes, err := k.OpenDisputes.Get(ctx)
	if err != nil {
		return err
	}
	openDisputes.Ids = append(openDisputes.Ids, disputeId)
	err = k.OpenDisputes.Set(ctx, openDisputes)
	if err != nil {
		return err
	}
	return nil
}

// Set DISPUTE
func (k Keeper) SetDispute(ctx sdk.Context, dispute types.Dispute) error {
	if err := k.AppendDisputeIdToOpenDisputeIds(ctx, dispute.DisputeId); err != nil {
		return err
	}
	return k.Disputes.Set(ctx, dispute.DisputeId, dispute)
}

// Set dispute status by dispute id
func (k Keeper) SetDisputeStatus(ctx sdk.Context, id uint64, status types.DisputeStatus) error {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}

	dispute.DisputeStatus = status
	return k.Disputes.Set(ctx, id, dispute)
}
