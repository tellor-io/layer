package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

// Set dispute status by dispute id
func (k Keeper) SetDisputeStatus(ctx sdk.Context, id uint64, status types.DisputeStatus) error {
	dispute := k.GetDisputeById(ctx, id)
	if dispute == nil {
		return types.ErrDisputeDoesNotExist
	}
	dispute.DisputeStatus = status
	k.SetDisputeById(ctx, id, *dispute)
	k.SetDisputeByReporter(ctx, *dispute)
	return nil
}
