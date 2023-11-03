package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// check if disputes can be removed due to expiration prior to commencing vote
func (k Keeper) CheckPrevoteDisputesForExpiration(ctx sdk.Context) []uint64 {
	openDisputes := k.GetOpenDisputeIds(ctx)

	var expiredDisputes []uint64
	var activeDisputes []uint64

	for _, disputeId := range openDisputes.Ids {
		// get dispute by id
		dispute := k.GetDisputeById(ctx, disputeId)
		if ctx.BlockTime().After(dispute.DisputeEndTime) && dispute.DisputeStatus == types.Prevote {
			// append to expired list
			expiredDisputes = append(expiredDisputes, disputeId)
		} else {
			// append to active list if not expired
			activeDisputes = append(activeDisputes, disputeId)
		}
	}
	// update active disputes list
	openDisputes.Ids = activeDisputes
	k.SetOpenDisputeIds(ctx, openDisputes)
	for _, disputeId := range expiredDisputes {
		// set dispute status to expired
		k.SetDisputeStatus(ctx, disputeId, types.Failed)
	}
	// return active list
	return activeDisputes
}
