package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) UpdateSnapshotLimit(goCtx context.Context, msg *types.MsgUpdateSnapshotLimit) (*types.MsgUpdateSnapshotLimitResponse, error) {
	if k.Keeper.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.Keeper.GetAuthority(), msg.Authority)
	}
	snapshotLimit, err := k.SnapshotLimit.Get(goCtx)
	if err != nil {
		return nil, err
	}
	snapshotLimit.Limit = msg.Limit
	if err := k.SnapshotLimit.Set(goCtx, snapshotLimit); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"snapshot_limit_updated",
			sdk.NewAttribute("limit", strconv.FormatUint(snapshotLimit.Limit, 10)),
		),
	})
	return &types.MsgUpdateSnapshotLimitResponse{}, nil
}
