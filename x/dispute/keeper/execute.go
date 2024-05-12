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
