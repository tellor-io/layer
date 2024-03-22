package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetValsetSigs(goCtx context.Context, req *types.QueryGetValsetSigsRequest) (*types.QueryGetValsetSigsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	sigs, err := k.GetValidatorSetSignaturesFromStorage(ctx, uint64(req.Timestamp))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get validator signatures")
	}

	// iterate through sigs and convert to hex + '0x'
	sigsHex := make([]string, len(sigs.Signatures))
	for i, sig := range sigs.Signatures {
		sigsHex[i] = "0x" + hex.EncodeToString(sig)
	}

	return &types.QueryGetValsetSigsResponse{
		Signatures: sigsHex,
	}, nil
}
