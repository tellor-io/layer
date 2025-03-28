package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetLastWithdrawalId(ctx context.Context, req *types.QueryGetLastWithdrawalIdRequest) (*types.QueryGetLastWithdrawalIdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	lastWithdrawalId, err := q.k.WithdrawalId.Get(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get last withdrawal id")
	}

	return &types.QueryGetLastWithdrawalIdResponse{WithdrawalId: lastWithdrawalId.Id}, nil
}
