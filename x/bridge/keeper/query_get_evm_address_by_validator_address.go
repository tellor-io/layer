package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetEvmAddressByValidatorAddress(ctx context.Context, req *types.QueryGetEvmAddressByValidatorAddressRequest) (*types.QueryGetEvmAddressByValidatorAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ethAddress, err := q.k.GetEVMAddressByOperator(ctx, req.ValidatorAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get eth address")
	}

	return &types.QueryGetEvmAddressByValidatorAddressResponse{EvmAddress: common.Bytes2Hex(ethAddress)}, nil
}
