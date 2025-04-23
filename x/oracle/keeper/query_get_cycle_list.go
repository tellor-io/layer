package keeper

import (
	"context"
	"encoding/hex"

	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) GetCycleList(ctx context.Context, req *types.QueryGetCycleListRequest) (*types.QueryGetCycleListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	cycleList, err := k.keeper.GetCyclelist(ctx)
	if err != nil {
		return nil, err
	}

	cycleListStr := make([]string, len(cycleList))
	for i, cycle := range cycleList {
		cycleListStr[i] = hex.EncodeToString(cycle)
	}

	return &types.QueryGetCycleListResponse{CycleList: cycleListStr}, nil
}
