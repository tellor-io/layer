package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	gomath "math"
	"math/big"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetDisputeByReporter assembles dispute key for reporter given the params and fetches the current dispute (the last generated one for this specific dispute)
// and returns the dispute object
func (k Keeper) GetDisputeByReporter(ctx sdk.Context, r oracletypes.MicroReport, c types.DisputeCategory) (types.Dispute, error) {
	key := []byte(k.ReporterKey(ctx, r, c))
	rng := collections.NewPrefixedPairRange[[]byte, uint64](key).Descending()
	// returns iterator for all the dispute ids sorted in descending order
	iter, err := k.Disputes.Indexes.DisputeByReporter.Iterate(ctx, rng)
	if err != nil {
		return types.Dispute{}, err
	}
	defer iter.Close()
	if !iter.Valid() {
		return types.Dispute{}, collections.ErrNotFound
	}
	// get the first dispute id
	id, err := iter.PrimaryKey()
	if err != nil {
		return types.Dispute{}, err
	}
	return k.Disputes.Get(ctx, id)
}

// NextDisputeId fetches an iterator for all disputes in descending order and increments the first returned dispute id by 1
func (k Keeper) NextDisputeId(ctx sdk.Context) uint64 {
	rng := new(collections.Range[uint64]).Descending()
	iter, err := k.Disputes.Iterate(ctx, rng)
	if err != nil {
		return 0
	}
	defer iter.Close()
	// for the first dispute id, return 1
	if !iter.Valid() {
		return 1
	}
	currentId, err := iter.Key()
	if err != nil {
		return 0
	}
	return currentId + 1
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
func (k Keeper) SetNewDispute(ctx sdk.Context, sender sdk.AccAddress, msg types.MsgProposeDispute) error {
	// validate report to make sure it exists
	exists, err := k.oracleKeeper.ValidateMicroReportExists(ctx, *msg.Report)
	ctx.Logger().Info("SetNewDispute", "exists", exists, "err", err)
	if err != nil {
		ctx.Logger().Error("SetNewDispute - validateMicroReportExists", "err", err)
		return err
	}
	if !exists {
		return fmt.Errorf("micro report does not exist")
	}
	disputeId := k.NextDisputeId(ctx)
	ctx.Logger().Info("SetNewDispute dispute id", "disputeId", disputeId)
	hashId := k.HashId(ctx, *msg.Report, msg.DisputeCategory)
	ctx.Logger().Info("SetNewDispute hash id", "hashId", hashId)
	// slash amount
	disputeFee, err := k.GetDisputeFee(ctx, *msg.Report, msg.DisputeCategory)
	if err != nil {
		return err
	}
	ctx.Logger().Info("SetNewDispute dispute fee", "disputeFee", disputeFee)
	if msg.Fee.Amount.GT(disputeFee) {
		msg.Fee.Amount = disputeFee
	}
	disputeFeeDec := math.LegacyNewDecFromInt(disputeFee)
	fivePercentDec := disputeFeeDec.Mul(math.LegacyNewDec(1)).Quo(math.LegacyNewDec(20))
	fivePercent := fivePercentDec.TruncateInt()
	dispute := types.Dispute{
		HashId:            hashId[:],
		DisputeId:         disputeId,
		DisputeCategory:   msg.DisputeCategory,
		DisputeStatus:     types.Prevote,
		DisputeStartTime:  ctx.BlockTime(),
		DisputeEndTime:    ctx.BlockTime().Add(ONE_DAY), // one day to fully pay fee
		DisputeStartBlock: uint64(ctx.BlockHeight()),
		DisputeRound:      1,
		SlashAmount:       disputeFee,
		// burn amount is calculated as 5% of dispute fee
		BurnAmount:      fivePercent,
		DisputeFee:      disputeFee,
		InitialEvidence: *msg.Report,
		FeeTotal:        msg.Fee.Amount,
		PrevDisputeIds:  []uint64{disputeId},
		Open:            true,
		BlockNumber:     uint64(ctx.BlockHeight()),
	}
	if err := k.DisputeFeePayer.Set(ctx, collections.Join(dispute.DisputeId, sender.Bytes()), types.PayerInfo{
		Amount:   msg.Fee.Amount,
		FromBond: msg.PayFromBond,
	}); err != nil {
		return err
	}
	// Pay the dispute fee
	if err := k.PayDisputeFee(ctx, sender, msg.Fee, msg.PayFromBond, dispute.HashId, true); err != nil {
		return err
	}

	// if the paid fee is equal to the slash amount, then slash validator and jail
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		if err := k.SlashAndJailReporter(ctx, dispute.InitialEvidence, dispute.DisputeCategory, dispute.HashId); err != nil {
			return err
		}
		// extend dispute end time by 3 days, 2 for voting and 1 to allow for more rounds
		dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
		dispute.DisputeStatus = types.Voting
		if err := k.SetStartVote(ctx, dispute.DisputeId); err != nil { // starting voting immediately
			return err
		}
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_dispute",
			sdk.NewAttribute("disputer", msg.Creator),
			sdk.NewAttribute("reporter", msg.Report.Reporter),
			sdk.NewAttribute("dispute_category", msg.DisputeCategory.String()),
			sdk.NewAttribute("total_fee", disputeFee.String()),
			sdk.NewAttribute("fee_paid", msg.Fee.Amount.String()),
			sdk.NewAttribute("pay_from_bond", strconv.FormatBool(msg.PayFromBond)),
			sdk.NewAttribute("dispute_id", strconv.FormatUint(disputeId, 10)),
			sdk.NewAttribute("value", msg.Report.Value),
			sdk.NewAttribute("query_type", msg.Report.QueryType),
			sdk.NewAttribute("query_id", hex.EncodeToString(msg.Report.QueryId)),
			sdk.NewAttribute("report_block_number", strconv.FormatUint(msg.Report.BlockNumber, 10)),
		),
	})
	return k.Disputes.Set(ctx, dispute.DisputeId, dispute)
}

// Slash and jail reporter
func (k Keeper) SlashAndJailReporter(ctx sdk.Context, report oracletypes.MicroReport, category types.DisputeCategory, hashId []byte) error {
	// flag aggregate report if necessary
	err := k.oracleKeeper.FlagAggregateReport(ctx, report)
	if err != nil {
		return err
	}
	reporterAddr := sdk.MustAccAddressFromBech32(report.Reporter)

	slashPercentageFixed6, jailDuration, err := GetSlashPercentageAndJailDuration(category)
	if err != nil {
		return err
	}
	reportPowerFixed6 := math.NewInt(int64(report.Power)).Mul(layertypes.PowerReduction)
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)
	reportPowerFixed6Dec := math.LegacyNewDecFromInt(reportPowerFixed6)
	slashPercentageFixed6Dec := math.LegacyNewDecFromInt(slashPercentageFixed6)
	slashAmountFixed6Dec := reportPowerFixed6Dec.Mul(slashPercentageFixed6Dec).Quo(powerReductionDec)
	slashAmountFixed6 := slashAmountFixed6Dec.TruncateInt()
	err = k.reporterKeeper.EscrowReporterStake(ctx, reporterAddr, report.Power, report.BlockNumber, slashAmountFixed6, report.QueryId, hashId)
	if err != nil {
		return err
	}
	return k.JailReporter(ctx, reporterAddr, jailDuration)
}

func (k Keeper) JailReporter(ctx context.Context, repAddr sdk.AccAddress, jailDuration uint64) error {
	// noop for major duration, reporter is removed from store so no need to jail
	return k.reporterKeeper.JailReporter(ctx, repAddr, jailDuration)
}

// Get percentage of slash amount based on category, returned as fixed6
func GetSlashPercentageAndJailDuration(category types.DisputeCategory) (math.Int, uint64, error) {
	switch category {
	case types.Warning:
		return math.NewInt(layertypes.PowerReduction.Int64()).QuoRaw(100), 0, nil // 1%
	case types.Minor:
		return math.NewInt(layertypes.PowerReduction.Int64()).QuoRaw(20), 600, nil // 5%
	case types.Major:
		return layertypes.PowerReduction, gomath.MaxInt64, nil // 100% and jails reporter for a year or 31536000 seconds. Will be deleted or unjailed depending on the results of the dispute
	default:
		return math.Int{}, 0, types.ErrInvalidDisputeCategory
	}
}

// Get dispute fee
func (k Keeper) GetDisputeFee(ctx sdk.Context, rep oracletypes.MicroReport, category types.DisputeCategory) (math.Int, error) {
	stake := layertypes.PowerReduction.MulRaw(int64(rep.Power))
	switch category {
	case types.Warning:
		// calculate 1 percent of bond
		stakeDec := math.LegacyNewDecFromInt(stake)
		feeDec := stakeDec.Mul(math.LegacyNewDec(1)).Quo(math.LegacyNewDec(100))
		return feeDec.TruncateInt(), nil
	case types.Minor:
		// calculate 5 percent of bond
		stakeDec := math.LegacyNewDecFromInt(stake)
		feeDec := stakeDec.Mul(math.LegacyNewDec(5)).Quo(math.LegacyNewDec(100))
		return feeDec.TruncateInt(), nil
	case types.Major:
		// calculate 100 percent of bond
		return stake, nil
	default:
		return math.Int{}, types.ErrInvalidDisputeCategory
	}
}

// Update existing dispute when conditions are met
// if dispute is unresolved then you can ingite another round.
// dispute round will have a new dispute id and the dispute.Round will be incremented.
// previous dispute will be closed.
func (k Keeper) AddDisputeRound(ctx sdk.Context, sender sdk.AccAddress, dispute types.Dispute, msg types.MsgProposeDispute) error {
	if dispute.DisputeStatus != types.Unresolved {
		return fmt.Errorf("can't start a new round for this dispute %d; dispute status %s", dispute.DisputeId, dispute.DisputeStatus)
	}

	if !dispute.Open {
		return fmt.Errorf("can't start a new round for this dispute %d; dispute closed", dispute.DisputeId)
	}
	// if dispute is not unresovled and dispute end time is before current block time then we can't update it
	if dispute.DisputeEndTime.Before(ctx.BlockTime()) {
		return fmt.Errorf("this dispute is expired, can't start new round %d", dispute.DisputeId)
	}

	if dispute.DisputeRound == 5 {
		return fmt.Errorf("can't start a new round for this dispute %d; max dispute rounds has been reached %d", dispute.DisputeId, dispute.DisputeRound)
	}
	// fee calculates a fee by scaling a base amount (fivePercent) exponentially based on the given round,
	// doubling for each successive round.
	fee := func(fivePercent math.Int, round int64) math.Int {
		base := new(big.Int).Exp(big.NewInt(2), big.NewInt(round), nil)
		return fivePercent.Mul(math.NewIntFromBigInt(base))
	}
	disputeSlashAmountDec := math.LegacyNewDecFromInt(dispute.SlashAmount)
	fivePercentDec := disputeSlashAmountDec.Mul(math.LegacyNewDec(1)).Quo(math.LegacyNewDec(20))
	fivePercent := fivePercentDec.TruncateInt()
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
	if err := k.PayDisputeFee(ctx, sender, msg.Fee, msg.PayFromBond, dispute.HashId, true); err != nil {
		return err
	}

	if err := k.CloseDispute(ctx, dispute.DisputeId); err != nil {
		return err
	}
	prevDisputeId := dispute.DisputeId
	dispute.BurnAmount = dispute.BurnAmount.Add(roundFee)
	dispute.FeeTotal = dispute.FeeTotal.Add(msg.Fee.Amount)
	disputeId := k.NextDisputeId(ctx)
	dispute.DisputeId = disputeId
	dispute.DisputeStatus = types.Voting // starting voting immediately
	dispute.DisputeStartTime = ctx.BlockTime()
	// add 3 days to block time
	dispute.DisputeEndTime = ctx.BlockTime().Add(THREE_DAYS)
	dispute.DisputeStartBlock = uint64(ctx.BlockHeight())
	dispute.DisputeRound++
	dispute.PrevDisputeIds = append(dispute.PrevDisputeIds, disputeId)

	err := k.Disputes.Set(ctx, dispute.DisputeId, dispute)
	if err != nil {
		return err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"added_dispute_round",
			sdk.NewAttribute("prev_dispute_id", strconv.FormatUint(prevDisputeId, 10)),
			sdk.NewAttribute("dispute_id", strconv.FormatUint(disputeId, 10)),
		),
	})
	return k.SetStartVote(ctx, dispute.DisputeId) // starting voting immediately
}

// creates a snapshot of total reporter power and total tips
func (k Keeper) SetBlockInfo(ctx context.Context, hashId []byte) error {
	tp, err := k.reporterKeeper.TotalReporterPower(ctx)
	if err != nil {
		return err
	}
	tips, err := k.oracleKeeper.GetTotalTips(ctx)
	if err != nil {
		return err
	}
	blockInfo := types.BlockInfo{
		TotalReporterPower: tp,
		TotalUserTips:      tips,
	}
	return k.BlockInfo.Set(ctx, hashId, blockInfo)
}

// close dispute by id
func (k Keeper) CloseDispute(ctx context.Context, id uint64) error {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}
	dispute.Open = false
	dispute.PendingExecution = false
	return k.Disputes.Set(ctx, id, dispute)
}

// gets all open disputes
func (k Keeper) GetOpenDisputes(ctx context.Context) ([]uint64, error) {
	iter, err := k.Disputes.Indexes.OpenDisputes.MatchExact(ctx, true)
	if err != nil {
		return nil, err
	}
	return iter.PrimaryKeys()
}
