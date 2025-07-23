package keeper

import (
	"context"
	"errors"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
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

	// the burnAmount starts at %5 of disputeFee, half of which is burned and the other half is distributed to the voters
	disputeBurnAmountDec := math.LegacyNewDecFromInt(dispute.BurnAmount)
	halfBurnAmountDec := disputeBurnAmountDec.Quo(math.LegacyNewDec(2))
	halfBurnAmount := halfBurnAmountDec.TruncateInt()
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
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		// burn half the burnAmount
		if !halfBurnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, halfBurnAmount))); err != nil {
				return err
			}
		}
		// stake the slashed tokens back into the bonded pool for the reporter
		if err := k.ReturnSlashedTokens(ctx, dispute); err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.GetReporter())
		if err != nil {
			return err
		}
		if err := k.reporterKeeper.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		// burn half the burnAmount
		if !halfBurnAmount.IsZero() {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, halfBurnAmount))); err != nil {
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
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, halfBurnAmount))); err != nil {
				return err
			}
		}
		// refund the reporters bond to the reporter plus the remaining disputeFee; goes to bonded pool
		ogSlashAmount := dispute.SlashAmount
		fivePercentDec := dispute.DisputeFee.ToLegacyDec().Quo(math.LegacyNewDec(20))
		disputeFeeMinusFivePercent := dispute.DisputeFee.Sub(fivePercentDec.TruncateInt())
		dispute.SlashAmount = dispute.SlashAmount.Add(disputeFeeMinusFivePercent)
		if err := k.ReturnSlashedTokens(ctx, dispute); err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(dispute.InitialEvidence.GetReporter())
		if err != nil {
			return err
		}
		if err := k.reporterKeeper.UpdateJailedUntilOnFailedDispute(ctx, reporterAddr); err != nil {
			return err
		}
		vote.Executed = true
		if err := k.Votes.Set(ctx, id, vote); err != nil {
			return err
		}
		dispute.SlashAmount = ogSlashAmount
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

func (k Keeper) RefundFailedDisputeFee(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, hashId []byte) error {
	fee := payerInfo.Amount

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, fee))
	if !payerInfo.FromBond {
		return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feePayer, coins)
	}

	return k.ReturnFeetoStake(ctx, hashId, fee)
}

func (k Keeper) RefundDisputeFee(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, disputeFeeTotal math.Int, hashId []byte, slashAmt math.Int) (math.Int, error) {
	// fee paid in rd 1
	payerFee := payerInfo.Amount
	payerFeeDec := payerFee.ToLegacyDec()
	// total fee rd 1
	totalFeeRd1 := math.LegacyNewDecFromInt(slashAmt)
	// total fee all rounds minus burn
	fivePercentDec := disputeFeeTotal.ToLegacyDec().Quo(math.LegacyNewDec(20))
	fivePercent := fivePercentDec.TruncateInt()
	totalFeeMinusBurnDec := disputeFeeTotal.Sub(fivePercent).ToLegacyDec()
	// power reductions
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)

	// (fee paid in rd1 / total fee rd 1)(total fee all rounds - burn)
	amtFixed12Dec := payerFeeDec.Mul(totalFeeMinusBurnDec).Mul(powerReductionDec).Quo(totalFeeRd1)

	amtFixed12 := amtFixed12Dec.TruncateInt()

	remainder := amtFixed12.Mod(layertypes.PowerReduction)

	amtFixed6 := amtFixed12.Quo(layertypes.PowerReduction)
	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amtFixed6))
	if !payerInfo.FromBond {
		return remainder, k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, feePayer, coins)
	}

	return remainder, k.ReturnFeetoStake(ctx, hashId, amtFixed6)
}

func (k Keeper) RewardReporterBondToFeePayers(ctx context.Context, feePayer sdk.AccAddress, payerInfo types.PayerInfo, totalFeesPaid, reporterBond math.Int) (math.Int, error) {
	bond := reporterBond
	totalFees := totalFeesPaid

	fee := payerInfo.Amount
	feeDec := math.LegacyNewDecFromInt(fee)
	bondDec := math.LegacyNewDecFromInt(bond)
	totalFeesDec := math.LegacyNewDecFromInt(totalFees)
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)
	amtFixed12Dec := feeDec.Mul(bondDec).Mul(powerReductionDec).Quo(totalFeesDec)
	amtFixed12 := amtFixed12Dec.TruncateInt()

	amtFixed6Dec := amtFixed12Dec.Quo(powerReductionDec)
	amtFixed6 := amtFixed6Dec.TruncateInt()
	if err := k.reporterKeeper.AddAmountToStake(ctx, feePayer, amtFixed6); err != nil {
		return math.Int{}, err
	}
	remainder := amtFixed12.Mod(layertypes.PowerReduction)
	return remainder, k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amtFixed6)))
}

func (k Keeper) GetSumOfAllGroupVotesAllRounds(ctx context.Context, id uint64) (math.Int, error) {
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}

	sumUsers := uint64(0)
	sumReporters := uint64(0)
	sumTeam := uint64(0)

	// process vote counts function
	processVoteCounts := func(voteCounts types.StakeholderVoteCounts) {
		sumUsers += voteCounts.Users.Support + voteCounts.Users.Against + voteCounts.Users.Invalid
		sumReporters += voteCounts.Reporters.Support + voteCounts.Reporters.Against + voteCounts.Reporters.Invalid
		sumTeam += voteCounts.Team.Support + voteCounts.Team.Against + voteCounts.Team.Invalid
	}

	// process current dispute
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return math.ZeroInt(), nil
	}
	processVoteCounts(voteCounts)

	// process previous disputes
	for _, roundId := range dispute.PrevDisputeIds {
		voteCounts, err := k.VoteCountsByGroup.Get(ctx, roundId)
		if err != nil {
			voteCounts = types.StakeholderVoteCounts{
				Users:     types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				Reporters: types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
				Team:      types.VoteCounts{Support: 0, Against: 0, Invalid: 0},
			}
		}
		processVoteCounts(voteCounts)
	}

	totalSum := math.NewInt(int64(sumUsers)).
		Add(math.NewInt(int64(sumReporters))).
		Add(math.NewInt(int64(sumTeam)))

	return totalSum, nil
}
