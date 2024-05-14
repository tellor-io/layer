package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetUserTipTotal(goCtx context.Context, req *types.QueryGetUserTipTotalRequest) (*types.QueryGetUserTipTotalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	tipper := sdk.MustAccAddressFromBech32(req.Tipper)
	totalTips, err := k.GetUserTips(goCtx, tipper)
	if err != nil {
		return nil, err
	}

	return &types.QueryGetUserTipTotalResponse{TotalTips: totalTips}, nil
}
