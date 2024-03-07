package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Get dispute by dispute id
func (k Keeper) GetDisputeById(ctx sdk.Context, id uint64) (*types.Dispute, error) {
	store := k.disputeStore(ctx)
	bz := store.Get(types.DisputeIdBytes(id))
	if bz == nil {
		return nil, types.ErrDisputeDoesNotExist.Wrapf("no dispute found with id %d", id)
	}
	var dispute types.Dispute
	err := k.cdc.Unmarshal(bz, &dispute)
	if err != nil {
		return nil, err
	}
	return &dispute, nil
}

// Get dispute by reporter key
func (k Keeper) GetDisputeByReporter(ctx sdk.Context, r types.MicroReport, c types.DisputeCategory) *types.Dispute {
	store := k.disputeStore(ctx)
	key := []byte(k.ReporterKey(ctx, r, c))
	bz := store.Get(key)
	if bz == nil {
		return nil
	}
	var dispute types.Dispute
	k.cdc.MustUnmarshal(bz, &dispute)
	return &dispute
}

// Get the dispute count from the store
func (k Keeper) GetDisputeCount(ctx sdk.Context) uint64 {
	store := k.disputeStore(ctx)
	byteKey := types.KeyPrefix(types.DisputeCountKey)
	bz := store.Get(byteKey)
	// Count doesn't exist: no disputes yet
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// Get open dispute ids from the store
func (k Keeper) GetOpenDisputeIds(ctx sdk.Context) (types.OpenDisputes, error) {
	store := k.disputeStore(ctx)
	bz := store.Get(types.OpenDisputeIdsKeyPrefix())
	if bz == nil {
		return types.OpenDisputes{}, nil
	}
	var ids types.OpenDisputes
	err := k.cdc.Unmarshal(bz, &ids)
	if err != nil {
		return types.OpenDisputes{}, err
	}
	return ids, nil
}

// Generate hash id
func (k Keeper) HashId(ctx sdk.Context, r types.MicroReport, c types.DisputeCategory) [32]byte {
	params := types.DisputeParams{
		Report:   &r,
		Category: c,
	}
	return sha256.Sum256(k.cdc.MustMarshal(&params))
}

// Make a reporter key by combining reporter address and a hash of dispute params
func (k Keeper) ReporterKey(ctx sdk.Context, r types.MicroReport, c types.DisputeCategory) string {
	return fmt.Sprintf("%s:%x", r.Reporter, k.HashId(ctx, r, c))
}

// Update the dispute count in the store
func (k Keeper) SetDisputeCount(ctx sdk.Context, count uint64) {
	store := k.disputeStore(ctx)
	byteKey := types.KeyPrefix(types.DisputeCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// Set dispute in the store by dispute id
func (k Keeper) SetDisputeById(ctx sdk.Context, id uint64, dispute types.Dispute) error {
	store := k.disputeStore(ctx)
	bz, err := k.cdc.Marshal(&dispute)
	if err != nil {
		return err
	}
	store.Set(types.DisputeIdBytes(id), bz)
	return nil
}

// Set dispute by reporter
func (k Keeper) SetDisputeByReporter(ctx sdk.Context, dispute types.Dispute) error {
	store := k.disputeStore(ctx)
	bz, err := k.cdc.Marshal(&dispute)
	if err != nil {
		return err
	}
	key := []byte(k.ReporterKey(ctx, dispute.ReportEvidence, dispute.DisputeCategory))
	store.Set(key, bz)
	return nil
}

// Set new dispute
func (k Keeper) SetNewDispute(ctx sdk.Context, msg types.MsgProposeDispute) error {
	disputeId := k.GetDisputeCount(ctx)
	hashId := k.HashId(ctx, *msg.Report, msg.DisputeCategory)
	// slash amount
	disputeFee, err := k.GetDisputeFee(ctx, msg.Report.Reporter, msg.DisputeCategory)
	if err != nil {
		return err
	}
	if disputeFee.IsZero() {
		return errors.New("error calculating dispute fee")
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
			Amount:       msg.Fee,
			FromBond:     msg.PayFromBond,
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
func (k Keeper) SlashAndJailReporter(ctx sdk.Context, report types.MicroReport, category types.DisputeCategory) error {
	reporterAddr := sdk.MustAccAddressFromBech32(report.Reporter)

	// k.Slash(ctx, sdk.ValAddress(accAddress), report.Power, k.GetSlashPercentage(category))
	var jailDuration int64
	switch category {
	case types.Warning:
		// jail for 0 seconds, forces validator to unjail manually
		jailDuration = 0
	case types.Minor:
		// jail for 10 minutes
		jailDuration = 600
	case types.Major:
		// TODO: jail duration should be until dispute ends
	default:
		return types.ErrInvalidDisputeCategory.Wrapf("wrong category selected: %s", category)
	}

	slashFactor, err := k.GetSlashPercentage(category)
	if err != nil {
		return err
	}
	amount := sdk.TokensFromConsensusPower(report.Power, sdk.DefaultPowerReduction)
	slashAmount := math.LegacyNewDecFromInt(amount).Mul(slashFactor)
	err = k.reporterKeeper.EscrowReporterStake(ctx, reporterAddr, report.BlockNumber, slashAmount.TruncateInt())
	if err != nil {
		return err
	}
	return k.reporterKeeper.JailReporter(ctx, reporterAddr, jailDuration)
}

// Store open dispute ids in the store
func (k Keeper) SetOpenDisputeIds(ctx sdk.Context, ids types.OpenDisputes) error {
	store := k.disputeStore(ctx)
	bz, err := k.cdc.Marshal(&ids)
	if err != nil {
		return err
	}
	store.Set(types.OpenDisputeIdsKeyPrefix(), bz)
	return nil
}

// Get percentage of slash amount based on category
func (k Keeper) GetSlashPercentage(category types.DisputeCategory) (math.LegacyDec, error) {
	switch category {
	case types.Warning:
		return math.LegacyNewDecWithPrec(1, 2), nil // 1%
	case types.Minor:
		return math.LegacyNewDecWithPrec(5, 2), nil // 5%
	case types.Major:
		return math.LegacyNewDecWithPrec(1, 0), nil // 100%
	default:
		return math.LegacyDec{}, errors.New("invalid dispute category")
	}
}

// Get dispute fee
func (k Keeper) GetDisputeFee(ctx sdk.Context, rep string, category types.DisputeCategory) (math.Int, error) {
	reporterAddr := sdk.MustAccAddressFromBech32(rep)
	reporter, err := k.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		return math.Int{}, err
	}

	stake := reporter.TotalTokens
	fee := math.ZeroInt()
	switch category {
	case types.Warning:
		// calculate 1 percent of bond
		fee = stake.MulRaw(1).QuoRaw(100)
	case types.Minor:
		// calculate 5 percent of bond
		fee = stake.MulRaw(5).QuoRaw(100)
	case types.Major:
		// calculate 100 percent of bond
		fee = stake
	}
	return fee, nil
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
	disputeId := k.GetDisputeCount(ctx)
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
	// How does second round of dispute fee work?
	// If fee is not paid then doubling the burnAmount means reducing the fee total?
	// Reducing the fee total means that feeTotal - burnAmount could be zero and the fee payers don't get anything from the feePaid or who gets what is not clear
	return nil
}

// remove dispute id from opendisputes after adding a new round
func (k Keeper) removeId(ctx sdk.Context, disputeId uint64) error {
	openDisputes, err := k.GetOpenDisputeIds(ctx)
	if err != nil {
		return err
	}
	for i, id := range openDisputes.Ids {
		if id == disputeId {
			openDisputes.Ids[i] = openDisputes.Ids[len(openDisputes.Ids)-1]
			openDisputes.Ids = openDisputes.Ids[:len(openDisputes.Ids)-1]
			err = k.SetOpenDisputeIds(ctx, openDisputes)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Add time to dispute end time
func (k Keeper) AddTimeToDisputeEndTime(ctx sdk.Context, id uint64, timeToAdd time.Duration) error {
	dispute, err := k.GetDisputeById(ctx, id)
	if err != nil {
		return err
	}
	dispute.DisputeEndTime = dispute.DisputeEndTime.Add(timeToAdd)
	err = k.SetDisputeById(ctx, dispute.DisputeId, *dispute)
	if err != nil {
		return err
	}
	return k.SetDisputeByReporter(ctx, *dispute)
}

// Append dispute id to open dispute ids
func (k Keeper) AppendDisputeIdToOpenDisputeIds(ctx sdk.Context, disputeId uint64) error {
	openDisputes, err := k.GetOpenDisputeIds(ctx)
	if err != nil {
		return err
	}
	openDisputes.Ids = append(openDisputes.Ids, disputeId)
	err = k.SetOpenDisputeIds(ctx, openDisputes)
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
	if err := k.SetDisputeByReporter(ctx, dispute); err != nil {
		return err
	}
	if err := k.SetDisputeById(ctx, dispute.DisputeId, dispute); err != nil {
		return err
	}
	k.SetDisputeCount(ctx, dispute.DisputeId+1)
	return nil
}

// Set dispute status by dispute id
func (k Keeper) SetDisputeStatus(ctx sdk.Context, id uint64, status types.DisputeStatus) error {
	dispute, err := k.GetDisputeById(ctx, id)
	if err != nil {
		return err
	}
	dispute.DisputeStatus = status
	err = k.SetDisputeById(ctx, id, *dispute)
	if err != nil {
		return err
	}

	return k.SetDisputeByReporter(ctx, *dispute)
}
