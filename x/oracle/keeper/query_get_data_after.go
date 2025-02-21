package keeper

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/tellor-io/layer/lib/metrics"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Querier) GetDataAfter(ctx context.Context, req *types.QueryGetDataAfterRequest) (*types.QueryGetDataAfterResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	qIdBz, err := utils.QueryBytesFromString(req.QueryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid queryId")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	defer telemetry.IncrCounterWithLabels([]string{"oracle_query_get_data_after"}, 1, []metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}, {Name: "query_id", Value: hex.EncodeToString(qIdBz)}})

	aggregate, timestamp, err := k.keeper.GetAggregateAfter(ctx, qIdBz, time.UnixMilli(int64(req.Timestamp)))
	if err != nil {
		return nil, err
	}

	timeUnix := timestamp.UnixMilli()

	return &types.QueryGetDataAfterResponse{
		Aggregate: aggregate,
		Timestamp: uint64(timeUnix),
	}, nil
}
