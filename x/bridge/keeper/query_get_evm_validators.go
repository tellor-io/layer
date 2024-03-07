package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetEvmAddressByValidatorAddress(goCtx context.Context, req *types.QueryGetEvmAddressByValidatorAddressRequest) (*types.QueryGetEvmAddressByValidatorAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	ethAddress, err := k.GetEVMAddressByOperator(ctx, req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get eth address")
	}

	return &types.QueryGetEvmAddressByValidatorAddressResponse{EvmAddress: ethAddress}, nil
}
