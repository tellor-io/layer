package keeper

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Get dispute by dispute id
func (k Keeper) GetDisputeById(ctx sdk.Context, id uint64) *types.Dispute {
	store := k.disputeStore(ctx)
	bz := store.Get(types.DisputeIdBytes(id))
	if bz == nil {
		return nil
	}
	var dispute types.Dispute
	k.cdc.MustUnmarshal(bz, &dispute)
	return &dispute
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
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// Get open dispute ids from the store
func (k Keeper) GetOpenDisputeIds(ctx sdk.Context) types.OpenDisputes {
	store := k.disputeStore(ctx)
	bz := store.Get(types.OpenDisputeIdsKeyPrefix())
	if bz == nil {
		return types.OpenDisputes{}
	}
	var ids types.OpenDisputes
	k.cdc.MustUnmarshal(bz, &ids)
	return ids
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
func (k Keeper) SetDisputeById(ctx sdk.Context, id uint64, dispute types.Dispute) {
	store := k.disputeStore(ctx)
	bz := k.cdc.MustMarshal(&dispute)
	store.Set(types.DisputeIdBytes(id), bz)
}

// Set dispute by reporter
func (k Keeper) SetDisputeByReporter(ctx sdk.Context, dispute types.Dispute) {
	store := k.disputeStore(ctx)
	bz := k.cdc.MustMarshal(&dispute)
	key := []byte(k.ReporterKey(ctx, dispute.ReportEvidence, dispute.DisputeCategory))
	store.Set(key, bz)
}

// Set new dispute
func (k Keeper) SetNewDispute(ctx sdk.Context, msg types.MsgProposeDispute) error {
	disputeId := k.GetDisputeCount(ctx)
	hashId := k.HashId(ctx, *msg.Report, msg.DisputeCategory)
	disputeFee := k.GetDisputeFee(ctx, msg.Report.Reporter, msg.DisputeCategory)
	if disputeFee.IsZero() {
		return fmt.Errorf("Error calculating dispute fee")
	}
	feeList := make([]types.PayerInfo, 0)

	if msg.Fee.Amount.GT(disputeFee) {
		msg.Fee.Amount = disputeFee
	}
	dispute := types.Dispute{
		HashId:            hashId[:],
		DisputeId:         disputeId,
		DisputeCategory:   msg.DisputeCategory,
		DisputeFee:        disputeFee,
		DisputeStatus:     types.Prevote,
		DisputeStartTime:  ctx.BlockTime(),
		DisputeEndTime:    ctx.BlockTime().Add(86400),
		DisputeStartBlock: ctx.BlockHeight(),
		DisputeRound:      1,
		SlashAmount:       disputeFee,
		// burn amount is calculated as 5% of dispute fee
		BurnAmount:     disputeFee.Mul(math.NewInt(1)).Quo(math.NewInt(20)),
		ReportEvidence: *msg.Report,
		FeePayers: append(feeList, types.PayerInfo{
			PayerAddress: msg.Creator,
			Amount:       msg.Fee,
			FromBond:     msg.PayFromBond,
		}),
		FeeTotal:       msg.Fee.Amount,
		PrevDisputeIds: make([]uint64, 0),
	}
	k.SetDispute(ctx, dispute)
	// Pay the dispute fee
	if err := k.PayDisputeFee(ctx, msg.Creator, msg.Fee, msg.PayFromBond); err != nil {
		return err
	}
	// if the paid fee is equal to the slash amount, then slash validator and jail
	if dispute.FeeTotal.Equal(dispute.SlashAmount) {
		k.SlashAndJailReporter(ctx, dispute.ReportEvidence, dispute.DisputeCategory)
	}

	return nil
}

// Slash and jail reporter
func (k Keeper) SlashAndJailReporter(ctx sdk.Context, report types.MicroReport, category types.DisputeCategory) {
	accAddress, err := sdk.AccAddressFromBech32(report.Reporter)
	if err != nil {
		panic(err)
	}
	k.Slash(ctx, sdk.ValAddress(accAddress), report.Power, k.GetSlashPercentage(category))
	switch category {
	case types.Warning:
		// jail for 0 seconds, forces validator to unjail manually
		k.JailValidatorUntil(ctx, sdk.ValAddress(accAddress), 0)
	case types.Minor:
		// jail for 10 minutes
		k.JailValidatorUntil(ctx, sdk.ValAddress(accAddress), 600)
	case types.Major:
		// no need to jail since validator will be removed from bonded pool so do nothing
	default:
		panic("invalid dispute category")
	}
}

// Store open dispute ids in the store
func (k Keeper) SetOpenDisputeIds(ctx sdk.Context, ids types.OpenDisputes) {
	store := k.disputeStore(ctx)
	bz := k.cdc.MustMarshal(&ids)
	store.Set(types.OpenDisputeIdsKeyPrefix(), bz)
}

// Get percentage of slash amount based on category
func (k Keeper) GetSlashPercentage(category types.DisputeCategory) math.LegacyDec {
	switch category {
	case types.Warning:
		return sdk.NewDecWithPrec(1, 2) // 1%
	case types.Minor:
		return sdk.NewDecWithPrec(5, 2) // 5%
	case types.Major:
		return sdk.NewDecWithPrec(1, 0) // 100%
	default:
		panic("invalid dispute category")
	}
}

// Get dispute fee
func (k Keeper) GetDisputeFee(ctx sdk.Context, reporter string, category types.DisputeCategory) math.Int {
	reporterAddr, err := sdk.AccAddressFromBech32(reporter)
	if err != nil {
		panic(err)
	}

	validator, found := k.stakingKeeper.GetValidator(ctx, sdk.ValAddress(reporterAddr))
	if !found {
		panic(fmt.Errorf("validator %s not found", reporter))
	}
	stake := validator.GetBondedTokens()
	fee := math.ZeroInt()
	onePercent := math.NewInt(1)
	fivePercent := math.NewInt(5)
	hundred := math.NewInt(100)
	switch category {
	case types.Warning:
		// calculate 1 percent of bond
		fee = stake.Mul(onePercent).Quo(hundred)
	case types.Minor:
		// calculate 5 percent of bond
		fee = stake.Mul(fivePercent).Quo(hundred)
	case types.Major:
		// calculate 100 percent of bond
		fee = stake
	}
	return fee
}

// Pay dispute fee
func (k Keeper) PayDisputeFee(ctx sdk.Context, sender string, fee sdk.Coin, fromBond bool) error {
	proposer, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return err
	}
	if fromBond {
		// pay fee from given validator
		err := k.PayFromBond(ctx, proposer, fee)
		if err != nil {
			return err
		}
	} else {
		err := k.PayFromAccount(ctx, proposer, fee)
		if err != nil {
			return err
		}
	}
	return nil
}

// Update existing dispute when conditions are met
func (k Keeper) AddDisputeRound(ctx sdk.Context, dispute types.Dispute) error {
	// if dispute is not unresovled and dispute end time is before current block time then we can't update it
	if dispute.DisputeStatus != types.Unresolved && dispute.DisputeEndTime.Before(ctx.BlockTime()) {
		return fmt.Errorf("this dispute is expired, can't start new round %d", dispute.DisputeId)
	}
	// if burnAmount is greater or equal to slashAmount then we can't update it
	if dispute.BurnAmount.GTE(dispute.SlashAmount) {
		return fmt.Errorf("this dispute has reached max rounds %d", dispute.DisputeId)
	}

	// increment burn amount by double
	dispute.BurnAmount = dispute.BurnAmount.Mul(math.NewInt(2))
	disputeId := k.GetDisputeCount(ctx)
	dispute.DisputeId = disputeId
	dispute.DisputeStatus = types.Voting // starting voting immediately
	dispute.DisputeStartTime = ctx.BlockTime()
	dispute.DisputeEndTime = ctx.BlockTime()
	dispute.DisputeStartBlock = ctx.BlockHeight()
	dispute.DisputeRound = dispute.DisputeRound + 1
	dispute.PrevDisputeIds = append(dispute.PrevDisputeIds, disputeId)

	k.SetDispute(ctx, dispute)
	// How does second round of dispute fee work?
	// If fee is not paid then doubling the burnAmount means reducing the fee total?
	// Reducing the fee total means that feeTotal - burnAmount could be zero and the fee payers don't get anything from the feePaid or who gets what is not clear
	return nil
}

// Add time to dispute end time
func (k Keeper) AddTimeToDisputeEndTime(ctx sdk.Context, id uint64, timeToAdd time.Duration) error {
	dispute := k.GetDisputeById(ctx, id)
	if dispute == nil {
		return types.ErrDisputeDoesNotExist
	}
	dispute.DisputeEndTime = dispute.DisputeEndTime.Add(timeToAdd)
	k.SetDisputeById(ctx, dispute.DisputeId, *dispute)
	k.SetDisputeByReporter(ctx, *dispute)
	return nil
}

// Append dispute id to open dispute ids
func (k Keeper) AppendDisputeIdToOpenDisputeIds(ctx sdk.Context, disputeId uint64) {
	openDisputes := k.GetOpenDisputeIds(ctx)
	openDisputes.Ids = append(openDisputes.Ids, disputeId)
	k.SetOpenDisputeIds(ctx, openDisputes)
}

// Set DISPUTE
func (k Keeper) SetDispute(ctx sdk.Context, dispute types.Dispute) {
	k.AppendDisputeIdToOpenDisputeIds(ctx, dispute.DisputeId)
	k.SetDisputeByReporter(ctx, dispute)
	k.SetDisputeById(ctx, dispute.DisputeId, dispute)
	k.SetDisputeCount(ctx, dispute.DisputeId+1)
}

// Set dispute status by dispute id
func (k Keeper) SetDisputeStatus(ctx sdk.Context, id uint64, status types.DisputeStatus) error {
	dispute := k.GetDisputeById(ctx, id)
	if dispute == nil {
		return types.ErrDisputeDoesNotExist
	}
	dispute.DisputeStatus = status
	k.SetDisputeById(ctx, id, *dispute)
	k.SetDisputeByReporter(ctx, *dispute)
	return nil
}
