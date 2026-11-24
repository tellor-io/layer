package keeper

import (
	"context"
	"errors"
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

	// Only users and reporters can claim a voter reward, including votes in prior rounds.
	totalVoterPower, err := k.GetSumOfUserAndReporterVotesAllRounds(ctx, id)
	if err != nil {
		return err
	}
	voterReward := math.ZeroInt()
	if totalVoterPower.IsPositive() {
		voterReward = dispute.BurnAmount.QuoRaw(2)
	}
	burnAmount := dispute.BurnAmount.Sub(voterReward)
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		if !burnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burnAmount))); err != nil {
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
		if !burnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burnAmount))); err != nil {
				return err
			}
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		if !burnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burnAmount))); err != nil {
				return err
			}
		}
		// refund the reporter's escrowed bond plus the dispute fee minus its 5% burn
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

func (k Keeper) RefundFailedDisputeFee(ctx context.Context, payer sdk.AccAddress, info types.PayerInfo, hashID []byte) error {
	// FeeRefund clamps a failed bonded payment to the available receipt.
	return k.returnPayerFee(ctx, payer, info, hashID, info.Amount)
}

func (k Keeper) RefundDisputeFee(ctx context.Context, payer sdk.AccAddress, info types.PayerInfo, quote, retainedFee math.Int, hashID []byte) (math.Int, error) {
	fee, dust := CalculateRefundAmount(info.Amount, quote, retainedFee)
	fee, err := k.backedFeeRefund(ctx, hashID, payer, info, fee)
	if err != nil {
		return math.Int{}, err
	}
	return dust, k.returnPayerFee(ctx, payer, info, hashID, fee)
}

func (k Keeper) returnPayerFee(ctx context.Context, payer sdk.AccAddress, info types.PayerInfo, hashID []byte, amount math.Int) error {
	if info.FromBond {
		// Empty rounds consume their empty receipt. An older shared receipt
		// with a positive balance belongs to a different contribution.
		if info.Amount.IsZero() {
			tracked, err := k.reporterKeeper.FeePaidFromStakeTotalByPayer(ctx, hashID, payer)
			if errors.Is(err, collections.ErrNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			if tracked.IsPositive() {
				return nil
			}
		}
		return k.ReturnFeetoStake(ctx, hashID, payer, amount)
	}
	if amount.IsZero() {
		return nil
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, payer, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amount)))
}

func (k Keeper) RewardReporterBondToFeePayers(ctx context.Context, payer sdk.AccAddress, info types.PayerInfo, quote, reporterBond math.Int) (math.Int, error) {
	principal, dust := CalculateReporterBondRewardAmount(info.Amount, quote, reporterBond)
	return dust, k.returnPayerPrincipal(ctx, payer, principal)
}

func (k Keeper) returnPayerPrincipal(ctx context.Context, payer sdk.AccAddress, principal math.Int) error {
	if principal.IsZero() {
		return nil
	}
	bonded, unbonded, err := k.reporterKeeper.AddAmountToStake(ctx, payer, principal)
	if err != nil {
		return err
	}
	return k.sendStakeReturnsToPools(ctx, bonded, unbonded)
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
