package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type VoterInfo struct {
	Voter sdk.AccAddress
	Power math.Int
	Share math.Int
}

// Execute the transfer of fee after the vote on a dispute is complete
func (k Keeper) ExecuteVote(ctx context.Context, id uint64) error {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}

	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}

	if vote.VoteResult != types.VoteResult_NO_TALLY && dispute.DisputeEndTime.Before(sdk.UnwrapSDKContext(ctx).BlockTime()) {
		dispute.DisputeStatus = types.Resolved
		dispute.Open = false
		if err := k.Disputes.Set(ctx, id, dispute); err != nil {
			return err
		}
	}

	if dispute.DisputeStatus != types.Resolved {
		return errors.New("can't execute, dispute not resolved")
	}

	if vote.Executed {
		return errors.New("vote already executed")
	}
	if vote.VoteResult == types.VoteResult_AGAINST || vote.VoteResult == types.VoteResult_NO_QUORUM_MAJORITY_AGAINST {
		if err := k.reconcileReporterWinFee(ctx, &dispute); err != nil {
			return err
		}
	}

	// Only user and reporter votes earn rewards. Compute one share and burn
	// the remainder so an odd allocation never leaves an unassigned loya.
	totalVoterPower, err := k.GetSumOfUserAndReporterVotesAllRounds(ctx, id)
	if err != nil {
		return err
	}
	voterReward := math.ZeroInt()
	if totalVoterPower.IsPositive() {
		voterReward = dispute.BurnAmount.QuoRaw(2)
	}
	burn := dispute.BurnAmount.Sub(voterReward)
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		if burn.IsPositive() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burn))); err != nil {
				return err
			}
		}
		// stake the slashed tokens back into the bonded pool for the reporter
		if err := k.ReturnSlashedTokens(ctx, dispute, math.ZeroInt()); err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.GetReporter())
		if err != nil {
			return err
		}
		if err := k.reporterKeeper.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, dispute.InitialEvidence.BlockNumber, StakeSnapshotOf(dispute.InitialEvidence), dispute.HashId); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		if burn.IsPositive() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burn))); err != nil {
				return err
			}
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		if burn.IsPositive() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burn))); err != nil {
				return err
			}
		}
		// Return the escrowed bond plus the backed first-round fee, less its allocation.
		disputeFeeMinusFivePercent := dispute.DisputeFee.Sub(dispute.DisputeFee.QuoRaw(20))
		if err := k.ReturnSlashedTokens(ctx, dispute, disputeFeeMinusFivePercent); err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.GetReporter())
		if err != nil {
			return err
		}
		if err := k.reporterKeeper.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr, dispute.InitialEvidence.BlockNumber, StakeSnapshotOf(dispute.InitialEvidence), dispute.HashId); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_NO_TALLY:
		return errors.New("vote hasn't been tallied yet")
	}
	dispute.VoterReward = voterReward
	dispute.PendingExecution = false
	if err := k.Disputes.Set(ctx, id, dispute); err != nil {
		return err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"dispute_executed",
			sdk.NewAttribute("dispute_id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("vote_result", vote.VoteResult.String()),
		),
	})
	return nil
}

// reconcileReporterWinFee replaces a legacy nominal round-one fee with its
// backing before paying the reporter, burning, or reserving voter rewards.
// The principal stays in the reporter escrow ledger; later fees are separate.
func (k Keeper) reconcileReporterWinFee(ctx context.Context, dispute *types.Dispute) error {
	if !dispute.DisputeFee.IsPositive() {
		return nil
	}
	firstRoundID := dispute.DisputeId
	if len(dispute.PrevDisputeIds) > 0 {
		firstRoundID = dispute.PrevDisputeIds[0]
	}
	_, received, err := k.roundOneReceipts(ctx, firstRoundID, dispute.HashId, false)
	if err != nil {
		return err
	}
	if received.LT(dispute.DisputeFee) {
		shortfall := dispute.DisputeFee.Sub(received)
		allocationReduction := dispute.DisputeFee.QuoRaw(20).Sub(received.QuoRaw(20))
		if dispute.BurnAmount.LT(allocationReduction) || dispute.FeeTotal.LT(shortfall) {
			return fmt.Errorf("inconsistent nominal fee accounting for dispute %d", dispute.DisputeId)
		}
		dispute.BurnAmount = dispute.BurnAmount.Sub(allocationReduction)
		dispute.FeeTotal = dispute.FeeTotal.Sub(shortfall)
		dispute.DisputeFee = received
	}
	return nil
}

func (k Keeper) RefundFailedDisputeFee(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, hashId []byte) error {
	fee := payerInfo.Amount

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, fee))
	if !payerInfo.FromBond {
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feePayer, coins)
	}

	return k.ReturnFeetoStake(ctx, hashId, feePayer, fee)
}

func (k Keeper) RefundDisputeFee(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, totalFeeRd1 math.Int, hashId []byte) (math.Int, error) {
	amtFixed6, remainder := CalculateRefundAmount(payerInfo.Amount, totalFeeRd1)

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amtFixed6))
	if !payerInfo.FromBond {
		return remainder, k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feePayer, coins)
	}

	actualFee, err := k.reporterKeeper.FeePaidFromStakeTotalByPayer(ctx, hashId, feePayer)
	if err != nil {
		return math.Int{}, err
	}
	amtFixed6 = CalculateActualStakeFeeRefundAmount(actualFee, payerInfo.Amount, amtFixed6)
	return remainder, k.ReturnFeetoStake(ctx, hashId, feePayer, amtFixed6)
}

func (k Keeper) RewardReporterBondToFeePayers(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, totalFeesPaid, reporterBond math.Int) (math.Int, error) {
	amtFixed6, remainder := CalculateReporterBondRewardAmount(payerInfo.Amount, totalFeesPaid, reporterBond)

	bondedAmt, unbondedAmt, err := k.reporterKeeper.AddAmountToStake(ctx, feePayer, amtFixed6)
	if err != nil {
		return math.Int{}, err
	}
	return remainder, k.sendStakeReturnsToPools(ctx, bondedAmt, unbondedAmt)
}

// GetSumOfUserAndReporterVotesAllRounds sums claim-eligible (user and reporter) vote
// power across all rounds of a dispute. Team votes are excluded because they cannot
// claim voter rewards.
func (k Keeper) GetSumOfUserAndReporterVotesAllRounds(ctx context.Context, id uint64) (math.Int, error) {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}

	// de-duplicate round ids (current id may already appear in PrevDisputeIds)
	roundIds := make([]uint64, 0, len(dispute.PrevDisputeIds)+1)
	seen := make(map[uint64]struct{})
	for _, roundId := range dispute.PrevDisputeIds {
		if _, ok := seen[roundId]; ok {
			continue
		}
		seen[roundId] = struct{}{}
		roundIds = append(roundIds, roundId)
	}
	if _, ok := seen[id]; !ok {
		roundIds = append(roundIds, id)
	}

	sumUsers := uint64(0)
	sumReporters := uint64(0)

	// process vote counts function
	processVoteCounts := func(voteCounts types.StakeholderVoteCounts) {
		sumUsers += voteCounts.Users.Support + voteCounts.Users.Against + voteCounts.Users.Invalid
		sumReporters += voteCounts.Reporters.Support + voteCounts.Reporters.Against + voteCounts.Reporters.Invalid
	}

	for _, roundId := range roundIds {
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, roundId)
		if err != nil {
			if !errors.Is(err, collections.ErrNotFound) {
				return math.Int{}, err
			}
			voteCounts = types.StakeholderVoteCounts{}
		}
		processVoteCounts(voteCounts)
	}

	totalSum := math.NewInt(int64(sumUsers)).
		Add(math.NewInt(int64(sumReporters)))

	return totalSum, nil
}
