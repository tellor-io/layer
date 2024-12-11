package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (q Querier) GetSnapshotLimit(goCtx context.Context, req *types.QueryGetSnapshotLimitRequest) (*types.QueryGetSnapshotLimitResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	snapshotLimit, err := q.k.SnapshotLimit.Get(ctx)
	if err != nil {
		return nil, status.Error(codes.NotFound, "snapshot limit not found")
	}

	return &types.QueryGetSnapshotLimitResponse{Limit: snapshotLimit.Limit}, nil
}
