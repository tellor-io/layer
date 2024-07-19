package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (q Querier) GetAttestationDataBySnapshot(goCtx context.Context, req *types.QueryGetAttestationDataBySnapshotRequest) (*types.QueryGetAttestationDataBySnapshotResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	snapshot, err := utils.QueryBytesFromString(req.Snapshot)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode snapshot %s", req.Snapshot)
	}

	snapshotData, err := q.k.AttestSnapshotDataMap.Get(ctx, snapshot)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("snapshot not found for snapshot %s", snapshot))
	}
	queryId := snapshotData.QueryId
	timestampTime := time.Unix(snapshotData.Timestamp, 0)

	aggReport, err := q.k.oracleKeeper.GetAggregateByTimestamp(ctx, queryId, timestampTime)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("aggregate not found for queryId %s and timestamp %s", queryId, timestampTime))
	}

	queryIdStr := hex.EncodeToString(queryId)
	timestampStr := strconv.FormatInt(snapshotData.Timestamp, 10)
	aggValueStr := aggReport.AggregateValue
	aggPowerStr := strconv.FormatInt(aggReport.ReporterPower, 10)
	checkpointStr := hex.EncodeToString(snapshotData.ValidatorCheckpoint)
	attestationTimestampStr := strconv.FormatInt(snapshotData.AttestationTimestamp, 10)
	previousReportTimestampStr := strconv.FormatInt(snapshotData.PrevReportTimestamp, 10)
	nextReportTimestampStr := strconv.FormatInt(snapshotData.NextReportTimestamp, 10)

	return &types.QueryGetAttestationDataBySnapshotResponse{QueryId: queryIdStr, Timestamp: timestampStr, AggregateValue: aggValueStr, AggregatePower: aggPowerStr, Checkpoint: checkpointStr, AttestationTimestamp: attestationTimestampStr, PreviousReportTimestamp: previousReportTimestampStr, NextReportTimestamp: nextReportTimestampStr}, nil
}
