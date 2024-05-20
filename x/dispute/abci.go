package dispute

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	return CheckPrevoteDisputesForExpiration(ctx, k)
}

func CheckPrevoteDisputesForExpiration(ctx context.Context, k keeper.Keeper) error {
	iter, err := k.Disputes.Indexes.OpenDisputes.MatchExact(ctx, true)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return err
		}
		if sdk.UnwrapSDKContext(ctx).BlockTime().After(dispute.DisputeEndTime) && dispute.DisputeStatus == types.Prevote {
			dispute.Open = false
			dispute.DisputeStatus = types.Failed
			if err := k.Disputes.Set(ctx, key, dispute); err != nil {
				return err
			}
		}
	}
	return nil
}
