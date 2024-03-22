package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetValsetByTimestamp(goCtx context.Context, req *types.QueryGetValsetByTimestampRequest) (*types.QueryGetValsetByTimestampResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	valset, err := k.GetBridgeValsetByTimestamp(ctx, uint64(req.Timestamp))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get eth address")
	}

	valsetArray := make([]*types.BridgeValidator, len(valset.BridgeValidatorSet))
	for i, val := range valset.BridgeValidatorSet {
		valsetArray[i] = &types.BridgeValidator{
			EthereumAddress: val.EthereumAddress,
			Power:           val.Power,
		}
	}

	return &types.QueryGetValsetByTimestampResponse{BridgeValidatorSet: valsetArray}, nil
}
