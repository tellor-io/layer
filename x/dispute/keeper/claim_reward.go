package keeper

import (
	"errors"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Pay fee from account
func (k Keeper) ClaimReward(ctx sdk.Context, addr sdk.AccAddress, id uint64) error {
	// check if dispute exists
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return err
	}

	if dispute.DisputeStatus != types.Resolved {
		return errors.New("can't execute, dispute not resolved")
	}

	// check if caller already claimed
	voterInfo, err := k.Voter.Get(ctx, collections.Join(id, addr.Bytes()))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		// not found, so must not have been claimed
	} else {
		if voterInfo.RewardClaimed {
			return errors.New("reward already claimed")
		}
	}

	reward, err := k.CalculateReward(ctx, addr, id)
	if err != nil {
		return err
	}
	if reward.IsZero() {
		return errors.New("reward is zero")
	}

	voterInfo.RewardClaimed = true
	if err := k.Voter.Set(ctx, collections.Join(id, addr.Bytes()), voterInfo); err != nil {
		return err
	}

	// send reward from this module to the address
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, reward))); err != nil {
		return err
	}

	return nil
}

// CalculateReward calculates the dispute reward for a given voter and disputeId
func (k Keeper) CalculateReward(ctx sdk.Context, addr sdk.AccAddress, id uint64) (math.Int, error) {
	k.Logger(ctx).Info("CalculateReward", "CalculateReward")
	k.Logger(ctx).Info("addr", "addr", addr)
	k.Logger(ctx).Info("id", "id", id)
	dispute, err := k.Disputes.Get(ctx, id)
	k.Logger(ctx).Info("dispute", "dispute", dispute)
	if err != nil {
		return math.Int{}, err
	}
	disputeVote, err := k.Votes.Get(ctx, id)
	k.Logger(ctx).Info("disputeVote", "disputeVote", disputeVote)
	if err != nil {
		return math.Int{}, err
	}
	if !disputeVote.Executed {
		return math.Int{}, errors.New("vote not executed")
	}

	addrReporterPower := math.ZeroInt()
	addrUserPower := math.ZeroInt()

	globalReporterPower := math.ZeroInt()
	globalUserPower := math.ZeroInt()

	for _, pastId := range dispute.PrevDisputeIds {
		k.Logger(ctx).Info("pastId", "pastId", pastId)
		pastVoterInfo, err := k.Voter.Get(ctx, collections.Join(pastId, addr.Bytes()))
		k.Logger(ctx).Info("pastVoterInfo", "pastVoterInfo", pastVoterInfo)
		k.Logger(ctx).Info("err", "err", err)
		if err == nil {
			// Voter info exists for this past dispute
			addrReporterPower = addrReporterPower.Add(pastVoterInfo.ReporterPower)
			k.Logger(ctx).Info("addrReporterPower", "addrReporterPower", addrReporterPower)
			userTips, err := k.GetUserTotalTips(ctx, addr, pastId)
			if err != nil {
				return math.Int{}, err
			}
			k.Logger(ctx).Info("userTips", "userTips", userTips)
			addrUserPower = addrUserPower.Add(userTips)
			k.Logger(ctx).Info("addrUserPower", "addrUserPower", addrUserPower)
		}

		// Get global vote counts for the past dispute
		pastVoteCounts, err := k.VoteCountsByGroup.Get(ctx, pastId)
		k.Logger(ctx).Info("pastVoteCounts", "pastVoteCounts", pastVoteCounts)
		k.Logger(ctx).Info("err", "err", err)
		if err != nil {
			return math.Int{}, err
		}
		// Add up the global power for each group
		globalReporterPower = globalReporterPower.Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Invalid))
		globalUserPower = globalUserPower.Add(math.NewIntFromUint64(pastVoteCounts.Users.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Users.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Users.Invalid))
		k.Logger(ctx).Info("globalReporterPower", "globalReporterPower", globalReporterPower)
		k.Logger(ctx).Info("globalUserPower", "globalUserPower", globalUserPower)
	}
	// nice way to handle zero division and zero votes
	totalGroups := int64(2)
	if globalReporterPower.IsZero() {
		k.Logger(ctx).Info("globalReporterPower", "globalReporterPower", globalReporterPower)
		globalReporterPower = math.NewInt(1)
		totalGroups--
	}
	if globalUserPower.IsZero() {
		k.Logger(ctx).Info("globalUserPower", "globalUserPower", globalUserPower)
		globalUserPower = math.NewInt(1)
		totalGroups--
	}
	if totalGroups == 0 {
		return math.Int{}, errors.New("no votes found")
	}

	// normalize powers
	powerReductionDec := math.LegacyNewDecFromInt(layer.PowerReduction)
	addrUserPowerDec := math.LegacyNewDecFromInt(addrUserPower)
	addrReporterPowerDec := math.LegacyNewDecFromInt(addrReporterPower)
	globalUserPowerDec := math.LegacyNewDecFromInt(globalUserPower)
	globalReporterPowerDec := math.LegacyNewDecFromInt(globalReporterPower)
	totalGroupsDec := math.LegacyNewDecFromInt(math.NewInt(totalGroups))
	disputeVoterRewardDec := math.LegacyNewDecFromInt(dispute.VoterReward)

	userPower := addrUserPowerDec.Mul(powerReductionDec).Quo(globalUserPowerDec)
	k.Logger(ctx).Info("userPower", "userPower", userPower)
	reporterPower := addrReporterPowerDec.Mul(powerReductionDec).Quo(globalReporterPowerDec)
	k.Logger(ctx).Info("reporterPower", "reporterPower", reporterPower)
	totalAccPower := userPower.Add(reporterPower)
	k.Logger(ctx).Info("totalAccPower", "totalAccPower", totalAccPower)
	rewardAcc := totalAccPower.Mul(disputeVoterRewardDec).Quo(totalGroupsDec.Mul(powerReductionDec))
	k.Logger(ctx).Info("rewardAcc", "rewardAcc", rewardAcc)

	return rewardAcc.TruncateInt(), nil
}
