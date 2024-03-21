package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// tally votes
func (k Keeper) Tally(ctx sdk.Context, ids []uint64) error {
	for _, id := range ids {
		if err := k.TallyVote(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Execute the transfer of fee after the vote on a dispute is complete
func (k Keeper) ExecuteVote(ctx sdk.Context, id uint64) error {
	dispute, err := k.GetDisputeById(ctx, id)
	if err != nil {
		return err
	}
	var voters []string
	for _, id := range dispute.PrevDisputeIds {
		v, err := k.GetVote(ctx, id)
		if err != nil {
			return err
		}
		voters = append(voters, v.Voters...)
	}
	vote, err := k.GetVote(ctx, id)
	if err != nil {
		return err
	}
	if vote.Executed || dispute.DisputeStatus != types.Resolved {
		ctx.Logger().Info("can't execute vote, reason either vote has already executed: %v, or dispute not resolved: %v", vote.Executed, dispute.DisputeStatus)
		return nil
	}
	// amount of dispute fee to return to fee payers or give to reporter
	disputeFeeMinusBurn := dispute.SlashAmount.Sub(dispute.BurnAmount)
	// the burnAmount starts at %5 of disputeFee, half of which is burned and the other half is distributed to the voters
	halfBurnAmount := dispute.BurnAmount.QuoRaw(2)
	voterReward := halfBurnAmount
	if len(voters) == 0 {
		// if no voters, burn the entire burnAmount
		halfBurnAmount = dispute.BurnAmount
		// non voters get nothing
		voterReward = math.ZeroInt()
	}
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		// distribute the voterRewardunt equally among the voters and transfer it to their accounts
		burnRemainder, err := k.RewardVoters(ctx, voters, voterReward)
		if err != nil {
			return err
		}
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount.Add(burnRemainder)))); err != nil {
			return err
		}
		// refund the remaining dispute fee to the fee payers according to their payment method
		if err := k.RefundDisputeFee(ctx, dispute.FeePayers, disputeFeeMinusBurn); err != nil {
			return err
		}
		// stake the slashed tokens back into the bonded pool for the reporter
		if err := k.ReturnSlashedTokens(ctx, dispute); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.SetVote(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		burnRemainder, err := k.RewardVoters(ctx, voters, voterReward)
		if err != nil {
			return err
		}
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount.Add(burnRemainder)))); err != nil {
			return err
		}
		// refund the remaining dispute fee to the fee payers according to their payment method
		if err := k.RefundDisputeFee(ctx, dispute.FeePayers, disputeFeeMinusBurn); err != nil {
			return err
		}
		// divide the reporters bond equally amongst the dispute fee payers and add it to the bonded pool
		if err := k.RewardReporterBondToFeePayers(ctx, dispute.FeePayers, dispute.SlashAmount); err != nil {
			return err
		}

		vote.Executed = true
		if err := k.SetVote(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		burnRemainder, err := k.RewardVoters(ctx, voters, voterReward)
		if err != nil {
			return err
		}
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount.Add(burnRemainder)))); err != nil {
			return err
		}
		// refund the reporters bond to the reporter plus the remaining disputeFee; goes to bonded pool
		dispute.SlashAmount = dispute.SlashAmount.Add(disputeFeeMinusBurn)
		if err := k.ReturnSlashedTokens(ctx, dispute); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.SetVote(ctx, id, vote); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ExecuteVotes(ctx sdk.Context, ids []uint64) error {
	for _, id := range ids {
		err := k.ExecuteVote(ctx, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// set disputes to resolved if adding rounds has been exhausted
// check if disputes can be removed due to expiration prior to commencing vote
func (k Keeper) CheckPrevoteDisputesForExpiration(ctx sdk.Context) ([]uint64, error) {
	openDisputes, err := k.GetOpenDisputeIds(ctx)
	if err != nil {
		return nil, err
	}
	var expiredDisputes []uint64 // disputes that failed to begin vote (ie fee unpaid in full)
	var activeDisputes []uint64

	for _, disputeId := range openDisputes.Ids {
		// get dispute by id
		dispute, err := k.GetDisputeById(ctx, disputeId)
		if err != nil {
			return nil, err
		}

		if ctx.BlockTime().After(dispute.DisputeEndTime) && dispute.DisputeStatus == types.Prevote {
			// append to expired list
			expiredDisputes = append(expiredDisputes, disputeId)
		} else {
			// append to active list if not expired
			activeDisputes = append(activeDisputes, disputeId)
		}
	}
	// update active disputes list
	openDisputes.Ids = activeDisputes
	err = k.SetOpenDisputeIds(ctx, openDisputes)
	if err != nil {
		return nil, err
	}
	for _, disputeId := range expiredDisputes {
		// set dispute status to expired
		err := k.SetDisputeStatus(ctx, disputeId, types.Failed)
		if err != nil {
			return nil, err
		}
	}
	// return active list
	return activeDisputes, nil
}
