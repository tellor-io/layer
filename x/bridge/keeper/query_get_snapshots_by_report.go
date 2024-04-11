package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetSnapshotsByReport(ctx context.Context, req *types.QueryGetSnapshotsByReportRequest) (*types.QueryGetSnapshotsByReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	// TODO: @tim is this correct? It's hashing over the hex encoded queryId
	key := crypto.Keccak256([]byte(hex.EncodeToString(req.QueryId) + fmt.Sprint(req.Timestamp)))
	snapshots, err := k.AttestSnapshotsByReportMap.Get(ctx, key)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("snapshots not found for queryId %x and timestamp %d", req.QueryId, req.Timestamp))
	}

	return &types.QueryGetSnapshotsByReportResponse{Snapshots: snapshots.Snapshots}, nil
}
