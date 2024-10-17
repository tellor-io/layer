package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

	// TODO: check if caller already claimed

	return nil
}

func (k Keeper) CalculateReward(ctx sdk.Context, addr sdk.AccAddress, id uint64) (math.Int, error) {

	dispute, err := k.Disputes.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	voteCounts, err := k.VoteCountsByGroup.Get(ctx, id)
	if err != nil {
		return math.Int{}, err
	}
	addrReporterPower := math.ZeroInt()
	addrTokenholderPower := math.ZeroInt()
	addrUserPower := math.ZeroInt()
	voterInfo, err := k.Voter.Get(ctx, collections.Join(id, addr.Bytes()))
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return math.Int{}, err
		}
		// not found, but could exist in past dispute rounds
	} else {
		// found in current dispute
		addrReporterPower = voterInfo.ReporterPower
		addrTokenholderPower = voterInfo.TokenholderPower
		addrUserPower, err = k.GetUserTotalTips(ctx, addr, dispute.BlockNumber)
		if err != nil {
			return math.Int{}, err
		}
	}

	globalReporterPower := math.NewIntFromUint64(voteCounts.Reporters.Support).Add(math.NewIntFromUint64(voteCounts.Reporters.Against)).Add(math.NewIntFromUint64(voteCounts.Reporters.Invalid))
	globalUserPower := math.NewIntFromUint64(voteCounts.Users.Support).Add(math.NewIntFromUint64(voteCounts.Users.Against)).Add(math.NewIntFromUint64(voteCounts.Users.Invalid))
	globalTokenholderPower := math.NewIntFromUint64(voteCounts.Tokenholders.Support).Add(math.NewIntFromUint64(voteCounts.Tokenholders.Against)).Add(math.NewIntFromUint64(voteCounts.Tokenholders.Invalid))

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
		globalReporterPower = globalReporterPower.Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Reporters.Invalid))
		globalUserPower = globalUserPower.Add(math.NewIntFromUint64(pastVoteCounts.Users.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Users.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Users.Invalid))
		globalTokenholderPower = globalTokenholderPower.Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Support)).
			Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Against)).Add(math.NewIntFromUint64(pastVoteCounts.Tokenholders.Invalid))
	}

// 	- dispRewardByAddrONE_ROUND = totalReward * (userTokens*1e6/totalTokensVoted + usrRep*1e6/totalRepVoted + usrTips*1e6/totalTipsVoted) * 1/3
// - dispRewards_TWO_ROUNDS = totalReward * [ (userTokens_r1 + userTokens_r2)*1e6 / (totalVoted_r1 + totalVoted_r2)*1e6 + / ... ) * 1/3 
	userPower := addrUserPower.Mul(math.NewInt(1e6)).Div(globalUserPower)
	reporterPower := addrReporterPower.Mul(math.NewInt(1e6)).Div(globalReporterPower)
	tokenholderPower := addrTokenholderPower.Mul(math.NewInt(1e6)).Div(globalTokenholderPower)
	totalAccPower := userPower.Add(reporterPower).Add(tokenholderPower)
	totalAccPower.Quo(math.NewInt(3))

	reward := 

	return math.Int{}, nil
}

// Pay fee from validator's bond can only be called by the validator itself
func (k Keeper) PayFromBond(ctx sdk.Context, reporterAddr sdk.AccAddress, fee sdk.Coin, hashId []byte) error {
	return k.reporterKeeper.FeefromReporterStake(ctx, reporterAddr, fee.Amount, hashId)
}

// Pay dispute fee
func (k Keeper) PayDisputeFee(ctx sdk.Context, proposer sdk.AccAddress, fee sdk.Coin, fromBond bool, hashId []byte) error {
	if fromBond {
		// pay fee from given validator
		err := k.PayFromBond(ctx, proposer, fee, hashId)
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

// return slashed tokens when reporter either wins dispute or dispute is invalid
func (k Keeper) ReturnSlashedTokens(ctx context.Context, dispute types.Dispute) error {
	err := k.reporterKeeper.ReturnSlashedTokens(ctx, dispute.SlashAmount, dispute.HashId)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, dispute.SlashAmount))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}

func (k Keeper) ReturnFeetoStake(ctx context.Context, hashId []byte, remainingAmt math.Int) error {
	err := k.reporterKeeper.FeeRefund(ctx, hashId, remainingAmt)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, remainingAmt))
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, coins)
}
