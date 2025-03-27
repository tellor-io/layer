package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetDepositClaimed(ctx context.Context, req *types.QueryGetDepositClaimedRequest) (*types.QueryGetDepositClaimedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	claimed, err := q.k.DepositIdClaimedMap.Get(ctx, req.DepositId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryGetDepositClaimedResponse{Claimed: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get deposit claimed")
	}

	return &types.QueryGetDepositClaimedResponse{Claimed: claimed.Claimed}, nil
}
