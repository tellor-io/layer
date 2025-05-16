package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

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

	aggReport, err := q.k.oracleKeeper.GetAggregateByTimestamp(ctx, queryId, snapshotData.Timestamp)
	var aggValueStr, aggPowerStr string
	if err != nil {
		// try to get no stake report
		noStakeReport, err := q.k.oracleKeeper.GetNoStakeReportByQueryIdTimestamp(ctx, queryId, snapshotData.Timestamp)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("aggregate / no stake report not found for queryId %s and timestamp %d", hex.EncodeToString(queryId), snapshotData.Timestamp))
		}
		aggValueStr = noStakeReport.Value
		aggPowerStr = "0"
	} else {
		aggValueStr = aggReport.AggregateValue
		aggPowerStr = strconv.FormatUint(aggReport.AggregatePower, 10)
	}

	queryIdStr := hex.EncodeToString(queryId)
	timestampStr := strconv.FormatUint(snapshotData.Timestamp, 10)
	checkpointStr := hex.EncodeToString(snapshotData.ValidatorCheckpoint)
	attestationTimestampStr := strconv.FormatUint(snapshotData.AttestationTimestamp, 10)
	previousReportTimestampStr := strconv.FormatUint(snapshotData.PrevReportTimestamp, 10)
	nextReportTimestampStr := strconv.FormatUint(snapshotData.NextReportTimestamp, 10)
	lastConsensusTimestampStr := strconv.FormatUint(snapshotData.LastConsensusTimestamp, 10)

	return &types.QueryGetAttestationDataBySnapshotResponse{
		QueryId:                 queryIdStr,
		Timestamp:               timestampStr,
		AggregateValue:          aggValueStr,
		AggregatePower:          aggPowerStr,
		Checkpoint:              checkpointStr,
		AttestationTimestamp:    attestationTimestampStr,
		PreviousReportTimestamp: previousReportTimestampStr,
		NextReportTimestamp:     nextReportTimestampStr,
		LastConsensusTimestamp:  lastConsensusTimestampStr,
	}, nil
}
