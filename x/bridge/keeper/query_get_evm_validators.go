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

	ethAddresses, _ := k.GetBridgeValidators(ctx)
	ethAddressesStr := make([]string, len(ethAddresses))
	for i, ethAddress := range ethAddresses {
		ethAddressesStr[i] = ethAddress.Hex()
	}

	return &types.QueryGetEvmValidatorsResponse{EthAddress: ethAddressesStr}, nil
}
