package keeper

import (
	"context"
	"errors"
	"strconv"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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
	// amount of dispute fee to return to fee payers or give to reporter
	disputeFeeMinusBurn := dispute.SlashAmount.Sub(dispute.BurnAmount)
	// the burnAmount starts at %5 of disputeFee, half of which is burned and the other half is distributed to the voters
	halfBurnAmount := dispute.BurnAmount.QuoRaw(2)
	voterReward := halfBurnAmount
	totalVoterPower, err := k.GetSumOfAllGroupVotesAllRounds(ctx, id)
	if err != nil {
		return err
	}
	if totalVoterPower.IsZero() {
		// if no voters, burn the entire burnAmount
		halfBurnAmount = dispute.BurnAmount
		// non voters get nothing
		voterReward = math.ZeroInt()
	}
	// why are we repeating logic in the switch statement?
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		// burn half the burnAmount
		if !halfBurnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount))); err != nil {
				return err
			}
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
		// burn half the burnAmount
		if !halfBurnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount))); err != nil {
				return err
			}
		}

		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		// burn half the burnAmount
		if !halfBurnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, halfBurnAmount))); err != nil {
				return err
			}
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
	case types.VoteResult_NO_TALLY:
		return errors.New("vote hasn't been tallied yet")
	}
	dispute.VoterReward = voterReward
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
	return k.BlockInfo.Remove(ctx, dispute.HashId)
}

func (k Keeper) RefundDisputeFee(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, totalFeesPaid, feeMinusBurn math.Int, hashId []byte) (math.LegacyDec, error) {
	fee := math.LegacyNewDecFromInt(payerInfo.Amount)
	totalFees := math.LegacyNewDecFromInt(totalFeesPaid)
	feeMinusBurnDec := math.LegacyNewDecFromInt(feeMinusBurn)
	amt := fee.Mul(feeMinusBurnDec).Quo(totalFees)

	remainder := amt.Sub(amt.TruncateDec())

	coins := sdk.NewCoins(sdk.NewCoin(layer.BondDenom, amt.TruncateInt()))
	if !payerInfo.FromBond {
		return remainder, k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feePayer, coins)
	}

	return remainder, k.ReturnFeetoStake(ctx, hashId, amt.TruncateInt())
}

func (k Keeper) RewardReporterBondToFeePayers(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, totalFeesPaid, reporterBond math.Int) (math.LegacyDec, error) {
	bond := math.LegacyNewDecFromInt(reporterBond)
	totalFees := math.LegacyNewDecFromInt(totalFeesPaid)

	fee := math.LegacyNewDecFromInt(payerInfo.Amount)
	amt := fee.Mul(bond).Quo(totalFees)

	if err := k.reporterKeeper.AddAmountToStake(ctx, feePayer, amt.TruncateInt()); err != nil {
		return math.LegacyDec{}, err
	}
	remainder := amt.Sub(amt.TruncateDec())
	return remainder, k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, amt.TruncateInt())))
}

func (k Keeper) GetSumOfAllGroupVotesAllRounds(ctx context.Context, id uint64) (math.Int, error) {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}

	sumUsers := uint64(0)
	sumReporters := uint64(0)
	sumTokenholders := uint64(0)
	sumTeam := uint64(0)

	// process vote counts function
	processVoteCounts := func(voteCounts types.StakeholderVoteCounts) {
		sumUsers += voteCounts.Users.Support + voteCounts.Users.Against + voteCounts.Users.Invalid
		sumReporters += voteCounts.Reporters.Support + voteCounts.Reporters.Against + voteCounts.Reporters.Invalid
		sumTokenholders += voteCounts.Tokenholders.Support + voteCounts.Tokenholders.Against + voteCounts.Tokenholders.Invalid
		sumTeam += voteCounts.Team.Support + voteCounts.Team.Against + voteCounts.Team.Invalid
	}

	// process current dispute
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	processVoteCounts(voteCounts)

	// process previous disputes
	for _, roundId := range dispute.PrevDisputeIds {
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, roundId)
		if err != nil {
			return math.Int{}, err
		}
		processVoteCounts(voteCounts)
	}

	totalSum := math.NewInt(int64(sumUsers)).
		Add(math.NewInt(int64(sumReporters))).
		Add(math.NewInt(int64(sumTokenholders))).
		Add(math.NewInt(int64(sumTeam)))

	return totalSum, nil
}
