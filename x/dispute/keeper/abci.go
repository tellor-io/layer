package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// tally votes
func (k Keeper) Tally(ctx sdk.Context, ids []uint64) {
	for _, id := range ids {
		k.TallyVote(ctx, id)
	}
}

// Execute the transfer of fee after the vote on a dispute is complete
func (k Keeper) ExecuteVote(ctx sdk.Context, id uint64) {
	dispute := k.GetDisputeById(ctx, id)
	if dispute == nil {
		return
	}
	var voters []string
	for _, id := range dispute.PrevDisputeIds {
		voters = append(voters, k.GetVote(ctx, id).Voters...)
	}
	vote := k.GetVote(ctx, id)
	if vote.Executed || dispute.DisputeStatus == types.Unresolved {
		return
	}
	disputeFeeMinusBurn := dispute.SlashAmount.Sub(dispute.BurnAmount)
	// the burnAmount %5 of disputeFee, half of which is burned and the other half is distributed to the voters
	burnCoins := dispute.BurnAmount.QuoRaw(2)
	voterReward := burnCoins
	if len(vote.Voters) == 0 {
		burnCoins = dispute.BurnAmount
		voterReward = sdk.ZeroInt()
	}
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(Denom, burnCoins))); err != nil {
			panic(err)
		}
		// divide the remaining burnAmount equally among the voters and transfer it to their accounts
		if err := k.RewardVoters(ctx, voters, voterReward); err != nil {
			panic(err)
		}
		// refund all fees to each dispute fee payer and restore validator bond/power
		// burn dispute fee then pay back the remaining dispute fee to the fee payers
		fromAcc, fromBond := k.SortPayerInfo(dispute.FeePayers)
		k.RefundDisputeFeeToAccount(ctx, fromAcc)
		k.RefundDisputeFeeToBond(ctx, fromBond)
		k.RefundToBond(ctx, dispute.ReportEvidence.Reporter, sdk.NewCoin(Denom, dispute.SlashAmount))
		vote.Executed = true
		k.SetVote(ctx, id, vote)
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(Denom, burnCoins))); err != nil {
			panic(err)
		}
		// divide the remaining burnAmount equally among the voters and transfer it to their accounts
		if err := k.RewardVoters(ctx, voters, voterReward); err != nil {
			panic(err)
		}
		// divide the reporters bond equally amongst the dispute fee payers and add it to the bonded pool
		reporterSlashAmount := dispute.SlashAmount.QuoRaw(int64(len(dispute.FeePayers)))
		for _, disputer := range dispute.FeePayers {
			k.RefundToBond(ctx, disputer.PayerAddress, sdk.NewCoin(Denom, reporterSlashAmount))
		}
		vote.Executed = true
		k.SetVote(ctx, id, vote)
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		// burn half the burnAmount
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(Denom, burnCoins))); err != nil {
			panic(err)
		}
		// divide the remaining burnAmount equally among the voters and transfer it to their accounts
		if err := k.RewardVoters(ctx, voters, voterReward); err != nil {
			panic(err)
		}
		// refund the reporters bond to the reporter plus the remaining disputeFee; goes to bonded pool
		k.RefundToBond(ctx, dispute.ReportEvidence.Reporter, sdk.NewCoin(Denom, dispute.SlashAmount.Add(disputeFeeMinusBurn)))
		vote.Executed = true
		k.SetVote(ctx, id, vote)
	default:
	}
}

func (k Keeper) ExecuteVotes(ctx sdk.Context, ids []uint64) {
	for _, id := range ids {
		k.ExecuteVote(ctx, id)
	}
}

// set disputes to resolved if adding rounds has been exhausted
// check if disputes can be removed due to expiration prior to commencing vote
func (k Keeper) CheckPrevoteDisputesForExpiration(ctx sdk.Context) []uint64 {
	openDisputes := k.GetOpenDisputeIds(ctx)
	var expiredDisputes []uint64 // disputes that failed to begin vote (ie fee unpaid in full)
	var activeDisputes []uint64

	for _, disputeId := range openDisputes.Ids {
		// get dispute by id
		dispute := k.GetDisputeById(ctx, disputeId)

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
	k.SetOpenDisputeIds(ctx, openDisputes)
	for _, disputeId := range expiredDisputes {
		// set dispute status to expired
		k.SetDisputeStatus(ctx, disputeId, types.Failed)
	}
	// return active list
	return activeDisputes
}
