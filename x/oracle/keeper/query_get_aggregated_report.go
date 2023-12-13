package keeper

import (
	"context"
	"encoding/hex"
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

	var aggregatedReport types.Aggregate
	store := k.AggregateStore(ctx)
	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		panic(err)
	}
	availableTimestamps := k.GetAvailableTimestampsByQueryId(ctx, queryId)
	if len(availableTimestamps.Timestamps) == 0 {
		return nil, fmt.Errorf("no available timestamps")
	}
	mostRecentTimestamp := availableTimestamps.Timestamps[len(availableTimestamps.Timestamps)-1]
	key := types.AggregateKey(queryId, mostRecentTimestamp)
	bz := store.Get(key)
	k.cdc.MustUnmarshal(bz, &aggregatedReport)
	return &types.QueryGetAggregatedReportResponse{Report: &aggregatedReport}, nil
}
