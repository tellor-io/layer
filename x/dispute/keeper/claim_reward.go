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
	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	disputeVote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	if !disputeVote.Executed {
		return math.Int{}, errors.New("vote not executed")
	}

	addrReporterPower := math.ZeroInt()
	addrTokenholderPower := math.ZeroInt()
	addrUserPower := math.ZeroInt()

	globalReporterPower := math.ZeroInt()
	globalUserPower := math.ZeroInt()
	globalTokenholderPower := math.ZeroInt()

	for _, pastId := range dispute.PrevDisputeIds {
		pastVoterInfo, err := k.Voter.Get(ctx, collections.Join(pastId, addr.Bytes()))
		if err == nil {
			// Voter info exists for this past dispute
			addrReporterPower = addrReporterPower.Add(pastVoterInfo.ReporterPower)
			addrTokenholderPower = addrTokenholderPower.Add(pastVoterInfo.TokenholderPower)
			userTips, err := k.GetUserTotalTips(ctx, addr, pastId)
			if err != nil {
				return math.Int{}, err
			}
			addrUserPower = addrUserPower.Add(userTips)
		}

		// Get global vote counts for the past dispute
		pastVoteCounts, err := k.VoteCountsByGroup.Get(ctx, pastId)
		if err != nil {
			return math.Int{}, err
		}
		// Add up the global power for each group
		globalReporterPower = globalReporterPower.Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Invalid))
		globalUserPower = globalUserPower.Add(math.NewIntFromUint64(pastVoteCounts.Users.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Users.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Users.Invalid))
		globalTokenholderPower = globalTokenholderPower.Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Invalid))
	}
	// nice way to handle zero division and zero votes
	totalGroups := int64(3)
	if globalReporterPower.IsZero() {
		globalReporterPower = math.NewInt(1)
		totalGroups--
	}
	if globalUserPower.IsZero() {
		globalUserPower = math.NewInt(1)
		totalGroups--
	}
	if globalTokenholderPower.IsZero() {
		globalTokenholderPower = math.NewInt(1)
		totalGroups--
	}
	if totalGroups == 0 {
		return math.Int{}, errors.New("no votes found")
	}

	// normalize powers
	powerReductionDec := math.LegacyNewDecFromInt(layer.PowerReduction)
	addrUserPowerDec := math.LegacyNewDecFromInt(addrUserPower)
	addrReporterPowerDec := math.LegacyNewDecFromInt(addrReporterPower)
	addrTokenholderPowerDec := math.LegacyNewDecFromInt(addrTokenholderPower)
	globalUserPowerDec := math.LegacyNewDecFromInt(globalUserPower)
	globalReporterPowerDec := math.LegacyNewDecFromInt(globalReporterPower)
	globalTokenholderPowerDec := math.LegacyNewDecFromInt(globalTokenholderPower)
	totalGroupsDec := math.LegacyNewDecFromInt(math.NewInt(totalGroups))
	disputeVoterRewardDec := math.LegacyNewDecFromInt(dispute.VoterReward)

	userPower := addrUserPowerDec.Mul(powerReductionDec).Quo(globalUserPowerDec)
	reporterPower := addrReporterPowerDec.Mul(powerReductionDec).Quo(globalReporterPowerDec)
	tokenholderPower := addrTokenholderPowerDec.Mul(powerReductionDec).Quo(globalTokenholderPowerDec)
	totalAccPower := userPower.Add(reporterPower).Add(tokenholderPower)
	rewardAcc := totalAccPower.Mul(disputeVoterRewardDec).Quo(totalGroupsDec.Mul(powerReductionDec))

	return rewardAcc.TruncateInt(), nil
}
