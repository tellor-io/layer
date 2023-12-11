package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
	return
}

func (k Keeper) RefundDisputeFeeToAccount(ctx sdk.Context, fromAcc []types.PayerInfo) error {
	var inputs []banktypes.Input
	var outputs []banktypes.Output

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	totalAmount := sdk.NewCoins()

	// Calculate total amount and prepare outputs
	for _, recipient := range fromAcc {
		burn := recipient.Amount.Amount.MulRaw(1).QuoRaw(20)
		recipient.Amount.Amount = recipient.Amount.Amount.Sub(burn)
		totalAmount = totalAmount.Add(sdk.NewCoins(recipient.Amount)...)
		outputs = append(outputs, banktypes.NewOutput(sdk.MustAccAddressFromBech32(recipient.PayerAddress), sdk.NewCoins(recipient.Amount)))
	}

	// Prepare input
	inputs = []banktypes.Input{banktypes.NewInput(moduleAddress, totalAmount)}

	// Perform the InputOutputCoins operation
	return k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}

func (k Keeper) RefundDisputeFeeToBond(ctx sdk.Context, fromBond []types.PayerInfo) error {
	for _, recipient := range fromBond {
		burn := recipient.Amount.Amount.MulRaw(1).QuoRaw(20)
		recipient.Amount.Amount = recipient.Amount.Amount.Sub(burn)
		if err := k.RefundToBond(ctx, recipient.PayerAddress, recipient.Amount); err != nil {
			panic(err)
		}
	}
	return nil
}

func (k Keeper) RewardVoters(ctx sdk.Context, voters []string, totalAmount math.Int) (math.Int, error) {
	if totalAmount.IsZero() {
		return totalAmount, nil
	}
	tokenDistribution, burnedRemainder := k.CalculateVoterShare(ctx, voters, totalAmount)
	totalAmount = totalAmount.Sub(burnedRemainder)
	var outputs []banktypes.Output
	for voter, share := range tokenDistribution {
		reward := sdk.NewCoins(sdk.NewCoin(Denom, share))
		outputs = append(outputs, banktypes.NewOutput(sdk.MustAccAddressFromBech32(voter), reward))
	}
	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	inputs := []banktypes.Input{banktypes.NewInput(moduleAddress, sdk.NewCoins(sdk.NewCoin(Denom, totalAmount)))}
	return burnedRemainder, k.bankKeeper.InputOutputCoins(ctx, inputs, outputs)
}
