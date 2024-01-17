package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetMedianValues(goCtx context.Context, req *types.QueryGetMedianValuesRequest) (*types.QueryGetMedianValuesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Process the query
	_ = ctx

	values := k.indexPriceCache.GetValidMedianPrices(k.marketParamConfig, time.Now())
	medianValues := make([]*types.MedianValues, 0, len(values))
	for i, value := range values {
		medianValues = append(medianValues, &types.MedianValues{
			MarketId: i,
			Price:    value,
		})
	}
	return &types.QueryGetMedianValuesResponse{MedianValues: medianValues}, nil
}
