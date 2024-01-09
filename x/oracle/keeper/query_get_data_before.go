package keeper

import (
	"context"
	"encoding/hex"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetDataBefore(goCtx context.Context, req *types.QueryGetDataBeforeRequest) (*types.QueryGetAggregatedReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	queryId, err := hex.DecodeString(req.QueryId)
	if err != nil {
		panic(err)
	}
	t := time.Unix(req.Timestamp, 0)
	report, err := k.getDataBefore(ctx, queryId, t)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetAggregatedReportResponse{Report: report}, nil
}
