package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (k Keeper) SortPayerInfo(feePayers []types.PayerInfo) (fromAcc, fromBond []types.PayerInfo) {
	for _, payer := range feePayers {
		if payer.FromBond {
			fromBond = append(fromBond, payer)
		} else {
			fromAcc = append(fromAcc, payer)
		}
	}
	return fromAcc, fromBond
}

func (k Keeper) RefundDisputeFeeToAccount(ctx sdk.Context, fromAcc []types.PayerInfo) error {
	var outputs []banktypes.Output

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	totalAmount := sdk.NewCoins()

	// Calculate total amount and prepare outputs
	for _, recipient := range fromAcc {
		burn := recipient.Amount.MulRaw(1).QuoRaw(20)
		recipient.Amount = recipient.Amount.Sub(burn)
		totalAmount = totalAmount.Add(sdk.NewCoins(sdk.NewCoin(layer.BondDenom, recipient.Amount))...)
		outputs = append(outputs, banktypes.NewOutput(sdk.MustAccAddressFromBech32(recipient.PayerAddress), sdk.NewCoins(sdk.NewCoin(layer.BondDenom, recipient.Amount))))
	}

	// Prepare input
	inputs := banktypes.NewInput(moduleAddress, totalAmount)

	// Perform the InputOutputCoins operation
	return k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}

func (k Keeper) RefundDisputeFeeToBond(ctx sdk.Context, fromBond []types.PayerInfo) error {
	// for every reporter refund dispute fee to their bond
	for _, recipient := range fromBond {
		burn := recipient.Amount.MulRaw(1).QuoRaw(20)
		recipient.Amount = recipient.Amount.Sub(burn)
		if err := k.reporterKeeper.ReturnSlashedTokens(ctx, recipient.PayerAddress, recipient.BlockNumber, recipient.Amount); err != nil {
			return err
		}
	}
	return nil
}
func (k Keeper) RewardReporterBondToFeePayers(ctx sdk.Context, feePayers []types.PayerInfo, reporterBond math.Int) error {
	totalFeesPaid := math.ZeroInt()
	for _, reporter := range feePayers {
		totalFeesPaid = totalFeesPaid.Add(reporter.Amount)
	}
	for _, reporter := range feePayers {
		amt := reporter.Amount.Quo(totalFeesPaid).Mul(reporterBond)
		if reporter.FromBond {
			if err := k.reporterKeeper.ReturnSlashedTokens(ctx, reporter.PayerAddress, reporter.BlockNumber, amt); err != nil {
				return err
			}
		} else {
			if err := k.reporterKeeper.AddAmountToStake(ctx, reporter.PayerAddress, amt); err != nil {
				return err
			}
		}
	}

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, reporterBond)))
}
func (k Keeper) RewardVoters(ctx sdk.Context, voters []string, totalAmount math.Int) (math.Int, error) {
	if totalAmount.IsZero() {
		return totalAmount, nil
	}
	tokenDistribution, burnedRemainder := k.CalculateVoterShare(ctx, voters, totalAmount)
	totalAmount = totalAmount.Sub(burnedRemainder)
	var outputs []banktypes.Output
	for voter, share := range tokenDistribution {
		if share.IsZero() {
			continue
		}
		reward := sdk.NewCoins(sdk.NewCoin(layer.BondDenom, share))
		outputs = append(outputs, banktypes.NewOutput(sdk.MustAccAddressFromBech32(voter), reward))
	}
	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	inputs := banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, totalAmount)))
	return burnedRemainder, k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}
