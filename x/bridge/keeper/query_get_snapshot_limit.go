package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

func (q Querier) GetSnapshotLimit(goCtx context.Context, req *types.QueryGetSnapshotLimitRequest) (*types.QueryGetSnapshotLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	snapshotLimit, err := q.k.SnapshotLimit.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetSnapshotLimitResponse{Limit: snapshotLimit.Limit}, nil
}
