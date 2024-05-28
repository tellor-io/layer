package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetEvmValidators(ctx context.Context, req *types.QueryGetEvmValidatorsRequest) (*types.QueryGetEvmValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valset, err := q.k.GetCurrentValidatorsEVMCompatible(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get current validators")
	}

	qValidatorSet := []*types.QueryBridgeValidator{}

	for _, val := range valset {
		qValidatorSet = append(qValidatorSet, &types.QueryBridgeValidator{
			EthereumAddress: common.Bytes2Hex(val.EthereumAddress),
			Power:           val.Power,
		})
	}

	return &types.QueryGetEvmValidatorsResponse{BridgeValidatorSet: qValidatorSet}, nil
}
