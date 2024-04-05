package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetAttestationsBySnapshot(goCtx context.Context, req *types.QueryGetAttestationsBySnapshotRequest) (*types.QueryGetAttestationsBySnapshotResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	snapshot := req.Snapshot

	ctx := sdk.UnwrapSDKContext(goCtx)

	attestations, err := k.SnapshotToAttestationsMap.Get(ctx, snapshot)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("attestations not found for snapshot %s", snapshot))
	}

	var attestationStringArray []string
	for _, attestation := range attestations.Attestations {
		attestationStringArray = append(attestationStringArray, hex.EncodeToString(attestation))
	}

	return &types.QueryGetAttestationsBySnapshotResponse{Attestations: attestationStringArray}, nil
}
