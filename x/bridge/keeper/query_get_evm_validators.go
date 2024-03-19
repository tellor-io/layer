package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetEvmValidators(goCtx context.Context, req *types.QueryGetEvmValidatorsRequest) (*types.QueryGetEvmValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	ethAddresses, err := k.GetCurrentValidatorsEVMCompatible(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get current validators")
	}
	ethAddressesStr := make([]string, len(ethAddresses))
	for i, ethAddresses := range ethAddresses {
		ethAddressesStr[i] = ethAddresses.EthereumAddress
	}

	return &types.QueryGetEvmValidatorsResponse{BridgeValidatorSet: ethAddresses}, nil
}
