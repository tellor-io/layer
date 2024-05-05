package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q Querier) GetSnapshotsByReport(ctx context.Context, req *types.QueryGetSnapshotsByReportRequest) (*types.QueryGetSnapshotsByReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	queryIdStr := req.QueryId
	timestampStr := req.Timestamp
	queryIdBytes, err := hex.DecodeString(queryIdStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid queryId %s", queryIdStr))
	}
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid timestamp %s", timestampStr))
	}
	timestampTime := time.Unix(timestampInt, 0)

	key := crypto.Keccak256([]byte(hex.EncodeToString(queryIdBytes) + fmt.Sprint(timestampTime.Unix())))
	snapshots, err := q.k.AttestSnapshotsByReportMap.Get(ctx, key)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("snapshots not found for queryId %s and timestamp %s", queryIdStr, timestampStr))
	}

	var snapshotStringArray []string
	for _, snapshot := range snapshots.Snapshots {
		snapshotStringArray = append(snapshotStringArray, hex.EncodeToString(snapshot))
	}

	return &types.QueryGetSnapshotsByReportResponse{Snapshots: snapshotStringArray}, nil
}
