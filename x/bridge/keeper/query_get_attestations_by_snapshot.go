package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetAttestationsBySnapshot(ctx context.Context, req *types.QueryGetAttestationsBySnapshotRequest) (*types.QueryGetAttestationsBySnapshotResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	snapshot, err := utils.QueryBytesFromString(req.Snapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode snapshot %s", req.Snapshot)
	}

	attestations, err := q.k.SnapshotToAttestationsMap.Get(ctx, snapshot)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("attestations not found for snapshot %s", req.Snapshot))
	}

	var attestationStringArray []string
	for _, attestation := range attestations.Attestations {
		attestationStringArray = append(attestationStringArray, hex.EncodeToString(attestation))
	}

	return &types.QueryGetAttestationsBySnapshotResponse{Attestations: attestationStringArray}, nil
}
