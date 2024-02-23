package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Querier) CurrentCyclelistQuery(goCtx context.Context, req *types.QueryCurrentCyclelistQueryRequest) (*types.QueryCurrentCyclelistQueryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	querydata, err := k.GetCurrentQueryInCycleList(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryCurrentCyclelistQueryResponse{Querydata: querydata}, nil
}
