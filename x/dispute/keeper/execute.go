package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		switch {
		case vote.VoteResult == types.VoteResult_INVALID:
			// refund all fees to each dispute fee payer and restore validator bond/power
		case vote.VoteResult == types.VoteResult_SUPPORT:
			// transfer fees(burnAmount) to voters/burncoin and transfer the validator bond and remaining dispute fee to dispute fee payers
		case vote.VoteResult == types.VoteResult_AGAINST:
			// transfer fees(burnAmount) to voters/burnCoin and add validator bond and dispute fee to bonded pool and set validator increased power
		default:
		}
	}
}
