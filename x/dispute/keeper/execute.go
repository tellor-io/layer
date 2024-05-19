package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

	vote, err := k.Votes.Get(ctx, id)
	if err != nil {
		return err
	}
	if vote.Executed || dispute.DisputeStatus != types.Resolved {
		k.Logger(ctx).Info("can't execute vote, reason either vote has already executed: %v, or dispute not resolved: %v", vote.Executed, dispute.DisputeStatus)
		return nil
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

func (k Keeper) RefundDisputeFee(ctx context.Context, feePayers []types.PayerInfo, remainingAmt math.Int, hashId []byte) error {
	var outputs []banktypes.Output

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	initialTotalAmount := math.ZeroInt()

	for _, recipient := range feePayers {
		initialTotalAmount = initialTotalAmount.Add(recipient.Amount)
	}

	accInputTotal := math.ZeroInt()
	// Calculate total amount and prepare outputs
	for _, recipient := range feePayers {
		amt := math.LegacyNewDecFromInt(recipient.Amount).Quo(math.LegacyNewDecFromInt(initialTotalAmount))
		amt = amt.MulInt(remainingAmt)

		coins := sdk.NewCoins(sdk.NewCoin(layer.BondDenom, amt.TruncateInt()))
		if !recipient.FromBond {
			accInputTotal = accInputTotal.Add(amt.TruncateInt())
			outputs = append(outputs, banktypes.NewOutput(sdk.MustAccAddressFromBech32(recipient.PayerAddress), coins))
		} else {
			if err := k.ReturnFeetoStake(ctx, sdk.MustAccAddressFromBech32(recipient.PayerAddress), hashId, amt.TruncateInt()); err != nil {
				return err
			}
		}

	}
	// Prepare input
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, accInputTotal)))

	// Perform the InputOutputCoins operation
	return k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}

func (k Keeper) RewardReporterBondToFeePayers(ctx context.Context, feePayers []types.PayerInfo, reporterBond math.Int) error {
	totalFeesPaid := math.ZeroInt()
	for _, feeInfo := range feePayers {
		totalFeesPaid = totalFeesPaid.Add(feeInfo.Amount)
	}
	// divvy up the reporter bond among the fee payers based how much they paid
	// paid it in as a stake in staking module
	for _, feeInfo := range feePayers {
		amt := feeInfo.Amount.Quo(totalFeesPaid).Mul(reporterBond)
		if err := k.reporterKeeper.AddAmountToStake(ctx, feeInfo.PayerAddress, amt); err != nil {
			return err
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, reporterBond)))
}
func (k Keeper) RewardVoters(ctx context.Context, voters []VoterInfo, totalAmount math.Int) (math.Int, error) {
	if totalAmount.IsZero() {
		return totalAmount, nil
	}
	tokenDistribution, burnedRemainder := k.CalculateVoterShare(ctx, voters, totalAmount)
	totalAmount = totalAmount.Sub(burnedRemainder)
	var outputs []banktypes.Output
	for _, v := range tokenDistribution {
		if v.Share.IsZero() {
			continue
		}
		reward := sdk.NewCoins(sdk.NewCoin(layer.BondDenom, v.Share))
		outputs = append(outputs, banktypes.NewOutput(v.Voter, reward))
	}
	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, totalAmount)))
	return burnedRemainder, k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}
