package keeper_test

import (
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func seedPowerThreshold(t *testing.T, k keeper.Keeper, ctx sdk.Context, threshold uint64) {
	t.Helper()
	require.NoError(t, k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{Index: 1}))
	require.NoError(t, k.ValidatorCheckpointIdxMap.Set(ctx, 1, types.CheckpointTimestamp{Timestamp: 100}))
	require.NoError(t, k.ValidatorCheckpointParamsMap.Set(ctx, 100, types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint"),
		ValsetHash:     []byte("valsetHash"),
		Timestamp:      100,
		PowerThreshold: threshold,
	}))
}

func TestGetLastConsensusTimestamp_EmptyMapReturnsZero(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	queryId := []byte("queryId")

	ts, err := k.GetLastConsensusTimestamp(sdkCtx, queryId)
	require.NoError(t, err)
	require.Equal(t, uint64(0), ts)
}

func TestGetLastConsensusTimestamp_ReturnsMapValue(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	queryId := []byte("queryId")
	require.NoError(t, k.SetLastConsensusTimestamp(sdkCtx, queryId, 12345))

	ts, err := k.GetLastConsensusTimestamp(sdkCtx, queryId)
	require.NoError(t, err)
	require.Equal(t, uint64(12345), ts)
}

func TestCreateSnapshot_NonConsensusEmptyMapSucceeds(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	timestamp := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(timestamp)
	queryId := []byte("queryId")

	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		AggregatePower: uint64(10),
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(time.Time{}, errors.New("no data before timestamp"))
	ok.On("GetTimestampAfter", sdkCtx, queryId, timestamp).Return(time.Time{}, errors.New("no data after timestamp"))

	require.NoError(t, k.ValidatorCheckpoint.Set(sdkCtx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(sdkCtx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)

	err := k.CreateSnapshot(sdkCtx, queryId, timestamp, false)
	require.NoError(t, err)

	attReq, err := k.AttestRequestsByHeightMap.Get(sdkCtx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	require.Len(t, attReq.Requests, 1)

	snapshot := attReq.Requests[0].Snapshot
	data, err := k.AttestSnapshotDataMap.Get(sdkCtx, snapshot)
	require.NoError(t, err)
	require.Equal(t, uint64(0), data.LastConsensusTimestamp)
}

func TestCreateSnapshot_NonConsensusUsesMapValue(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	timestamp := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(timestamp)
	queryId := []byte("queryId")
	mapTs := uint64(999)

	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "5000",
		AggregatePower: uint64(10),
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(timestamp.Add(-time.Hour), nil)
	ok.On("GetTimestampAfter", sdkCtx, queryId, timestamp).Return(time.Time{}, errors.New("no data after timestamp"))

	require.NoError(t, k.ValidatorCheckpoint.Set(sdkCtx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(sdkCtx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)
	require.NoError(t, k.SetLastConsensusTimestamp(sdkCtx, queryId, mapTs))

	err := k.CreateSnapshot(sdkCtx, queryId, timestamp, true)
	require.NoError(t, err)

	attReq, err := k.AttestRequestsByHeightMap.Get(sdkCtx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	data, err := k.AttestSnapshotDataMap.Get(sdkCtx, attReq.Requests[0].Snapshot)
	require.NoError(t, err)
	require.Equal(t, mapTs, data.LastConsensusTimestamp)
}

func TestRequestAttestations_NoPriorSnapshotSucceeds(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	timestamp := time.UnixMilli(1_000_000)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(timestamp)
	queryId := []byte("queryId")
	creator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "10",
		AggregatePower: uint64(10),
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, timestamp).Return(time.Time{}, errors.New("no data"))
	ok.On("GetTimestampAfter", sdkCtx, queryId, timestamp).Return(time.Time{}, errors.New("no data"))

	require.NoError(t, k.ValidatorCheckpoint.Set(sdkCtx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(sdkCtx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)

	_, err := msgServer.RequestAttestations(sdkCtx, &types.MsgRequestAttestations{
		Creator:   creator.String(),
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: strconv.FormatInt(timestamp.UnixMilli(), 10),
	})
	require.NoError(t, err)
}

func TestRequestAttestations_DoesNotRewindLastConsensusMap(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	oldTs := time.UnixMilli(1_000)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(oldTs)
	queryId := []byte("queryId")
	creator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	latestConsensus := uint64(9_000)

	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(oldTs.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: "10",
		AggregatePower: uint64(200),
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId, oldTs).Return(time.Time{}, errors.New("no data"))
	ok.On("GetTimestampAfter", sdkCtx, queryId, oldTs).Return(time.Time{}, errors.New("no data"))

	require.NoError(t, k.ValidatorCheckpoint.Set(sdkCtx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(sdkCtx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)
	require.NoError(t, k.SetLastConsensusTimestamp(sdkCtx, queryId, latestConsensus))

	_, err := msgServer.RequestAttestations(sdkCtx, &types.MsgRequestAttestations{
		Creator:   creator.String(),
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: strconv.FormatInt(oldTs.UnixMilli(), 10),
	})
	require.NoError(t, err)

	got, err := k.LastConsensusTimestampByQueryId.Get(sdkCtx, queryId)
	require.NoError(t, err)
	require.Equal(t, latestConsensus, got)

	requests, err := k.AttestRequestsByHeightMap.Get(sdkCtx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	require.Len(t, requests.Requests, 1)
	snapshotData, err := k.AttestSnapshotDataMap.Get(sdkCtx, requests.Requests[0].Snapshot)
	require.NoError(t, err)
	require.Equal(t, latestConsensus, snapshotData.LastConsensusTimestamp)
}

func TestCreateNewReportSnapshots_GetTimestampBeforeFailureReturns(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timestamp := sdkCtx.BlockTime()
	timestampPlus1 := timestamp.Add(time.Second)
	queryId1 := []byte("queryId1")
	queryId2 := []byte("queryId2")

	ok.On("GetAggregatedReportsByHeight", ctx, uint64(0)).Return([]oracletypes.Aggregate{
		{Height: 0, QueryId: queryId1, AggregateValue: "5000", AggregatePower: uint64(100)},
		{Height: 0, QueryId: queryId2, AggregateValue: "5000", AggregatePower: uint64(100)},
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, queryId1, timestampPlus1).Return(time.Time{}, errors.New("no data")).Once()

	require.NoError(t, k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)

	err := k.CreateNewReportSnapshots(ctx)
	require.Error(t, err)
	ok.AssertNotCalled(t, "GetTimestampBefore", mock.Anything, queryId2, mock.Anything)
}

func TestCreateNewReportSnapshots_UpdatesMapPastSnapshotLimit(t *testing.T) {
	k, _, _, ok, _, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timestamp := sdkCtx.BlockTime()
	timestampPlus1 := timestamp.Add(time.Second)
	require.NoError(t, k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: 1}))

	snapshotted := []byte("queryIdSnapshotted")
	overflowConsensus := []byte("queryIdOverflowConsensus")
	overflowWeak := []byte("queryIdOverflowWeak")

	ok.On("GetAggregatedReportsByHeight", ctx, uint64(0)).Return([]oracletypes.Aggregate{
		{Height: 0, QueryId: snapshotted, AggregateValue: "5000", AggregatePower: uint64(150)},
		{Height: 0, QueryId: overflowConsensus, AggregateValue: "5000", AggregatePower: uint64(150)},
		{Height: 0, QueryId: overflowWeak, AggregateValue: "5000", AggregatePower: uint64(10)},
	}, nil)
	ok.On("GetTimestampBefore", sdkCtx, snapshotted, timestampPlus1).Return(timestamp, nil).Once()
	ok.On("GetTimestampBefore", sdkCtx, overflowConsensus, timestampPlus1).Return(timestamp, nil).Once()
	ok.On("GetTimestampBefore", sdkCtx, overflowWeak, timestampPlus1).Return(timestamp, nil).Once()
	ok.On("GetTimestampBefore", sdkCtx, snapshotted, timestamp).Return(time.Time{}, errors.New("no data"))
	ok.On("GetTimestampAfter", ctx, snapshotted, timestamp).Return(time.Time{}, errors.New("no data"))
	ok.On("GetAggregateByTimestamp", ctx, snapshotted, uint64(timestamp.UnixMilli())).Return(oracletypes.Aggregate{
		QueryId:        snapshotted,
		AggregateValue: "5000",
		AggregatePower: uint64(150),
	}, nil)

	require.NoError(t, k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{Checkpoint: []byte("checkpoint")}))
	require.NoError(t, k.BridgeValset.Set(ctx, types.BridgeValidatorSet{
		BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("validator1"), Power: 100}},
	}))
	seedPowerThreshold(t, k, sdkCtx, 100)

	err := k.CreateNewReportSnapshots(ctx)
	require.NoError(t, err)

	attReq, err := k.AttestRequestsByHeightMap.Get(ctx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	require.Len(t, attReq.Requests, 1)
	ok.AssertNotCalled(t, "GetAggregateByTimestamp", mock.Anything, overflowConsensus, mock.Anything)
	ok.AssertNotCalled(t, "GetAggregateByTimestamp", mock.Anything, overflowWeak, mock.Anything)

	got, err := k.LastConsensusTimestampByQueryId.Get(sdkCtx, snapshotted)
	require.NoError(t, err)
	require.Equal(t, uint64(timestamp.UnixMilli()), got)
	got, err = k.LastConsensusTimestampByQueryId.Get(sdkCtx, overflowConsensus)
	require.NoError(t, err)
	require.Equal(t, uint64(timestamp.UnixMilli()), got)
	_, err = k.LastConsensusTimestampByQueryId.Get(sdkCtx, overflowWeak)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func TestGetLastConsensusTimestamp_NotFoundIsZero(t *testing.T) {
	k, _, _, _, _, _, _, ctx := setupKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_, err := k.LastConsensusTimestampByQueryId.Get(sdkCtx, []byte("missing"))
	require.ErrorIs(t, err, collections.ErrNotFound)
}
