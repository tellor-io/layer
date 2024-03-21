package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetCurrentTip(goCtx context.Context, req *types.QueryGetCurrentTipRequest) (*types.QueryGetCurrentTipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// req.QueryData = regtypes.Remove0xPrefix(req.QueryData)
	queryId, err := utils.QueryIDFromDataString(req.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid query data")
	}
	tips, err := k.GetQueryTip(ctx, queryId)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetCurrentTipResponse{Tips: &types.Tips{
		QueryData: req.QueryData, // TODO: avoid returning the same data as the request
		Amount:    tips,
	}}, nil
}
