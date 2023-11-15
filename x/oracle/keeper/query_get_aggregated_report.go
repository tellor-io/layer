package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetAggregatedReport(goCtx context.Context, req *types.QueryGetAggregatedReportRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if req.BlockNumber == 0 {
		req.BlockNumber = ctx.BlockHeight()
	}
	var aggregatedReport types.Aggregate
	store := k.AggregateStore(ctx)
	bz := store.Get([]byte(fmt.Sprintf("%s-%d", req.QueryId, req.BlockNumber)))
	k.cdc.MustUnmarshal(bz, &aggregatedReport)
	return &types.QueryGetAggregatedReportResponse{Report: &aggregatedReport}, nil
}
