package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

type RefundTo struct {
	validator string
	fee       sdk.Coin
}

// Execute the transfer of fee after the vote on a dispute is complete
func (k Keeper) ExecuteVote(ctx sdk.Context, ids []uint64) {

	for _, id := range ids {
		dispute := k.GetDisputeById(ctx, id)
		if dispute == nil {
			return
		}
		vote := k.GetVote(ctx, id)

		disputeFee := dispute.SlashAmount
		// TODO: check if already executed!
		switch vote.VoteResult {
		case types.VoteResult_INVALID:
			// refund all fees to each dispute fee payer and restore validator bond/power
			fromAcc, fromBond := k.SortPayerInfo(dispute.FeePayers)
			k.RefundDisputeFeeToAccount(ctx, fromAcc)
			k.RefundDisputeFeeToBond(ctx, fromBond)
			k.RefundToBond(ctx, dispute.ReportEvidence.Reporter, sdk.NewCoin(sdk.DefaultBondDenom, disputeFee))
		case types.VoteResult_SUPPORT:
			// transfer fees(burnAmount) to voters/burncoin and transfer the validator bond and remaining dispute fee to dispute fee payers
		case types.VoteResult_AGAINST:
			// transfer fees(burnAmount) to voters/burnCoin and add validator bond and dispute fee to bonded pool and set validator increased power
		default:
		}

	}
}

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
		k.RefundToBond(ctx, recipient.PayerAddress, recipient.Amount)
	}
	return nil
}
