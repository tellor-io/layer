package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetUserTipTotal(goCtx context.Context, req *types.QueryGetUserTipTotalRequest) (*types.QueryGetUserTipTotalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	tipper := sdk.MustAccAddressFromBech32(req.Tipper)

	var totalTips types.UserTipTotal
	if req.QueryData == "" {
		totalTips = k.GetUserTips(ctx, tipper)
		return &types.QueryGetUserTipTotalResponse{TotalTips: &totalTips}, nil
	}

	queryId, err := utils.QueryIDFromDataString(req.QueryData)
	if err != nil {
		return nil, err
	}

	// TODO: figure out query id here
	tip, err := k.Tips.Get(ctx, collections.Join(queryId, tipper.Bytes()))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return &types.QueryGetUserTipTotalResponse{TotalTips: &totalTips}, nil
		}
		return nil, err
	}
	totalTips.Total = sdk.NewCoin(types.DefaultBondDenom, tip)

	return &types.QueryGetUserTipTotalResponse{TotalTips: &totalTips}, nil
}
