package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetEvmValidators(ctx context.Context, req *types.QueryGetEvmValidatorsRequest) (*types.QueryGetEvmValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ethAddresses, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get current validators")
	}

	return &types.QueryGetEvmValidatorsResponse{BridgeValidatorSet: ethAddresses}, nil
}
