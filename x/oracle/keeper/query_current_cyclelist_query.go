package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) CurrentCyclelistQuery(goCtx context.Context, req *types.QueryCurrentCyclelistQueryRequest) (*types.QueryCurrentCyclelistQueryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	return &types.QueryCurrentCyclelistQueryResponse{Querydata: k.GetCurrentQueryInCycleList(ctx)}, nil
}
