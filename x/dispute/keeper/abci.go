package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
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
	var voters []VoterInfo
	for _, id := range dispute.PrevDisputeIds {
		iter, err := k.Voter.Indexes.VotersById.MatchExact(ctx, id)
		if err != nil {
			return err
		}

		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			key, err := iter.PrimaryKey()
			if err != nil {
				return err
			}
			v, err := k.Voter.Get(ctx, key)
			if err != nil {
				return err
			}
			voters = append(voters, VoterInfo{Voter: key.K2(), Power: v.VoterPower, Share: math.ZeroInt()})

		}
	}
	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}
	if vote.Executed || dispute.DisputeStatus != types.Resolved {
		k.Logger(ctx).Info("can't execute vote, reason either vote has already executed: %v, or dispute not resolved: %v", vote.Executed, dispute.DisputeStatus)
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
		if err := k.RefundDisputeFee(ctx, dispute.FeePayers, disputeFeeMinusBurn, dispute.HashId); err != nil {
			return err
		}
		// stake the slashed tokens back into the bonded pool for the reporter
		if err := k.ReturnSlashedTokens(ctx, dispute); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
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
		if err := k.RefundDisputeFee(ctx, dispute.FeePayers, disputeFeeMinusBurn, dispute.HashId); err != nil {
			return err
		}
		// divide the reporters bond equally amongst the dispute fee payers and add it to the bonded pool
		if err := k.RewardReporterBondToFeePayers(ctx, dispute.FeePayers, dispute.SlashAmount); err != nil {
			return err
		}

		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
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
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	}
	return k.BlockInfo.Remove(ctx, dispute.HashId)
}

func (k Keeper) ExecuteVotes(ctx context.Context, ids []uint64) error {
	for _, id := range ids {
		if err := k.ExecuteVote(ctx, id); err != nil {
			return err
		}

	}
	return nil
}

// set disputes to resolved if adding rounds has been exhausted
// check if disputes can be removed due to expiration prior to commencing vote
func (k Keeper) CheckPrevoteDisputesForExpiration(ctx context.Context) ([]uint64, error) {
	openDisputes, err := k.OpenDisputes.Get(ctx)
	if err != nil {
		return nil, err
	}
	var expiredDisputes []uint64 // disputes that failed to begin vote (ie fee unpaid in full)
	var activeDisputes []uint64

	for _, disputeId := range openDisputes.Ids {
		// get dispute by id
		dispute, err := k.Disputes.Get(ctx, disputeId)
		if err != nil {
			return nil, err
		}

		if sdk.UnwrapSDKContext(ctx).BlockTime().After(dispute.DisputeEndTime) && dispute.DisputeStatus == types.Prevote {
			// append to expired list
			expiredDisputes = append(expiredDisputes, disputeId)
		} else {
			// append to active list if not expired
			activeDisputes = append(activeDisputes, disputeId)
		}
	}
	// update active disputes list
	openDisputes.Ids = activeDisputes
	err = k.OpenDisputes.Set(ctx, openDisputes)
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
