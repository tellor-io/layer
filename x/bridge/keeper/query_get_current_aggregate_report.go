package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCurrentAggregateReport(goCtx context.Context, req *types.QueryGetCurrentAggregateReportRequest) (*types.QueryGetCurrentAggregateReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	queryIdBytes, err := hex.DecodeString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query id")
	}
	aggregate, timestamp := k.oracleKeeper.GetCurrentAggregateReport(ctx, queryIdBytes)
	if aggregate == nil {
		return nil, status.Error(codes.NotFound, "aggregate not found")
	}
	timeUnix := timestamp.Unix()

	return &types.QueryGetCurrentAggregateReportResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
