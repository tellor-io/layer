package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetValsetSigs(ctx context.Context, req *types.QueryGetValsetSigsRequest) (*types.QueryGetValsetSigsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	sigs, err := q.k.GetValidatorSetSignaturesFromStorage(ctx, uint64(req.Timestamp))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator signatures")
	}

	// iterate through sigs and convert to hex + '0x'
	sigsHex := make([]string, len(sigs.Signatures))
	for i, sig := range sigs.Signatures {
		sigsHex[i] = common.Bytes2Hex(sig)
	}

	return &types.QueryGetValsetSigsResponse{
		Signatures: sigsHex,
	}, nil
}
