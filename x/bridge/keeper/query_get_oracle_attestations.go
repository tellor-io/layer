package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetOracleAttestations(goCtx context.Context, req *types.QueryGetOracleAttestationsRequest) (*types.QueryGetOracleAttestationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	sigs, err := k.GetOracleAttestationsFromStorage(ctx, req.QueryId, uint64(req.Timestamp))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get oracle attestations")
	}

	// iterate through sigs and convert to hex + '0x'
	sigsHex := make([]string, len(sigs.Attestations))
	for i, sig := range sigs.Attestations {
		sigsHex[i] = "0x" + hex.EncodeToString(sig)
	}

	return &types.QueryGetOracleAttestationsResponse{
		Attestations: sigsHex,
	}, nil
}
