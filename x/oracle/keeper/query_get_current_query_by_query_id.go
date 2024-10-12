package keeper

import (
	"context"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetCurrentQueryByQueryId(ctx context.Context, req *types.QueryGetCurrentQueryByQueryIdRequest) (*types.QueryGetCurrentQueryByQueryIdResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qId, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, err
	}

	query, err := k.keeper.CurrentQuery(ctx, qId)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetCurrentQueryByQueryIdResponse{Query: &query}, nil
}
