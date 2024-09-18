package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValsetByTimestamp(ctx context.Context, req *types.QueryGetValsetByTimestampRequest) (*types.QueryGetValsetByTimestampResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	valset, err := q.k.GetBridgeValsetByTimestamp(ctx, req.Timestamp)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get eth address")
	}

	valsetArray := make([]*types.QueryBridgeValidator, len(valset.BridgeValidatorSet))
	for i, val := range valset.BridgeValidatorSet {
		ethAddr := common.BytesToAddress(val.EthereumAddress)
		valsetArray[i] = &types.QueryBridgeValidator{
			EthereumAddress: ethAddr.Hex(),
			Power:           val.Power,
		}
	}

	return &types.QueryGetValsetByTimestampResponse{BridgeValidatorSet: valsetArray}, nil
}
