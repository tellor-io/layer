package keeper_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	math "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestMsgRequestAttestations(t *testing.T) {
	k, _, _, ok, _, sk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	msgServer := keeper.NewMsgServerImpl(k)
	require.NotNil(t, msgServer)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.SetValsetCheckpointDomainSeparator(sdkCtx)

	// empty message
	response, err := msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{})
	require.ErrorContains(t, err, "invalid")
	require.Nil(t, response)

	// bad queryId
	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   "abcd1234",
		QueryId:   "z",
		Timestamp: "1",
	})
	require.ErrorContains(t, err, "invalid")
	require.Nil(t, response)

	// bad timestamp
	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   "abcd1234",
		QueryId:   "abcd1234",
		Timestamp: "z",
	})
	require.ErrorContains(t, err, "invalid")
	require.Nil(t, response)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(60 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(40 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}

	evmAddresses := make([]types.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}
	sk.On("GetAllValidators", ctx).Return(validators, nil)
	result, err := k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	queryId := []byte("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce499")
	timestampTime := time.Unix(1000, 0)
	aggReport := oracletypes.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "10",
		AggregatePower:    uint64(10),
		AggregateReporter: creatorAddr.String(),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId, uint64(timestampTime.UnixMilli())).Return(aggReport, nil)
	err = k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	ok.On("GetTimestampBefore", ctx, queryId, timestampTime).Return(timestampTime.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", ctx, queryId, timestampTime).Return(timestampTime.Add(1*time.Hour), nil)
	ok.On("GetCurrentAggregateReport", ctx, queryId).Return(&aggReport, timestampTime, nil)
	snapshotKey := crypto.Keccak256([]byte(hex.EncodeToString(queryId) + fmt.Sprint(timestampTime.UnixMilli())))
	snapshot := []byte("snapshot")
	err = k.AttestSnapshotsByReportMap.Set(ctx, snapshotKey, types.AttestationSnapshots{
		Snapshots: [][]byte{snapshot},
	})
	require.NoError(t, err)
	snapshotData := types.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("checkpoint"),
		AttestationTimestamp:   uint64(timestampTime.UnixMilli()),
		PrevReportTimestamp:    uint64(timestampTime.Add(-1 * time.Hour).UnixMilli()),
		NextReportTimestamp:    uint64(0),
		QueryId:                queryId,
		Timestamp:              uint64(timestampTime.UnixMilli()),
		LastConsensusTimestamp: uint64(timestampTime.Add(-2 * time.Hour).UnixMilli()),
	}
	err = k.AttestSnapshotDataMap.Set(ctx, snapshot, snapshotData)
	require.NoError(t, err)

	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   creatorAddr.String(),
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: strconv.FormatInt(timestampTime.UnixMilli(), 10),
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	expectedAttestationTimestamp := sdkCtx.BlockTime()

	// retrieve newly created snapshot & data
	snapshots, err := k.AttestSnapshotsByReportMap.Get(ctx, snapshotKey)
	require.NoError(t, err)
	require.Equal(t, len(snapshots.Snapshots), 2)
	require.Equal(t, snapshots.Snapshots[0], snapshot)
	snapshot2 := snapshots.Snapshots[1]
	snapshotData2, err := k.AttestSnapshotDataMap.Get(ctx, snapshot2)
	require.NoError(t, err)
	require.Equal(t, snapshotData2.LastConsensusTimestamp, uint64(timestampTime.Add(-2*time.Hour).UnixMilli()))
	require.Equal(t, snapshotData2.PrevReportTimestamp, uint64(timestampTime.Add(-1*time.Hour).UnixMilli()))
	require.Equal(t, snapshotData2.NextReportTimestamp, uint64(timestampTime.Add(1*time.Hour).UnixMilli()))
	require.Equal(t, snapshotData2.QueryId, queryId)
	require.Equal(t, snapshotData2.Timestamp, uint64(timestampTime.UnixMilli()))
	require.Equal(t, snapshotData2.ValidatorCheckpoint, []byte("checkpoint"))
	require.Equal(t, snapshotData2.AttestationTimestamp, uint64(expectedAttestationTimestamp.UnixMilli()))
}

func TestMsgRequestAttestations_DedupesDuplicateSnapshotAtSameHeight(t *testing.T) {
	k, _, _, ok, _, sk, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.SetValsetCheckpointDomainSeparator(sdkCtx)

	err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: 1})
	require.NoError(t, err)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2
	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(60 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(40 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}
	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)
	}
	sk.On("GetAllValidators", ctx).Return(validators, nil)
	_, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	queryId := []byte("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce499")
	timestampTime := time.Unix(1000, 0)
	aggReport := oracletypes.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "10",
		AggregatePower:    uint64(10),
		AggregateReporter: creatorAddr.String(),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId, uint64(timestampTime.UnixMilli())).Return(aggReport, nil)
	ok.On("GetTimestampBefore", ctx, queryId, timestampTime).Return(timestampTime.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", ctx, queryId, timestampTime).Return(timestampTime.Add(1*time.Hour), nil)
	ok.On("GetCurrentAggregateReport", ctx, queryId).Return(&aggReport, timestampTime, nil)
	err = k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	snapshotKey := crypto.Keccak256([]byte(hex.EncodeToString(queryId) + fmt.Sprint(timestampTime.UnixMilli())))
	snapshot := []byte("snapshot")
	err = k.AttestSnapshotsByReportMap.Set(ctx, snapshotKey, types.AttestationSnapshots{
		Snapshots: [][]byte{snapshot},
	})
	require.NoError(t, err)
	err = k.AttestSnapshotDataMap.Set(ctx, snapshot, types.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("checkpoint"),
		AttestationTimestamp:   uint64(timestampTime.UnixMilli()),
		PrevReportTimestamp:    uint64(timestampTime.Add(-1 * time.Hour).UnixMilli()),
		NextReportTimestamp:    uint64(0),
		QueryId:                queryId,
		Timestamp:              uint64(timestampTime.UnixMilli()),
		LastConsensusTimestamp: uint64(timestampTime.Add(-2 * time.Hour).UnixMilli()),
	})
	require.NoError(t, err)

	msg := &types.MsgRequestAttestations{
		Creator:   creatorAddr.String(),
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: strconv.FormatInt(timestampTime.UnixMilli(), 10),
	}
	_, err = msgServer.RequestAttestations(ctx, msg)
	require.NoError(t, err)
	_, err = msgServer.RequestAttestations(ctx, msg)
	require.NoError(t, err)

	requests, err := k.AttestRequestsByHeightMap.Get(ctx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	require.Len(t, requests.Requests, 1)
}

func TestMsgRequestAttestations_EnforcesExactSnapshotLimit(t *testing.T) {
	k, _, _, ok, _, sk, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.SetValsetCheckpointDomainSeparator(sdkCtx)

	err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: 1})
	require.NoError(t, err)

	operatorAddr1 := testOperatorAddr1
	operatorAddr2 := testOperatorAddr2
	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(60 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: operatorAddr1,
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(40 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: operatorAddr2,
		},
	}
	for _, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)
	}
	sk.On("GetAllValidators", ctx).Return(validators, nil)
	_, err = k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)

	err = k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	timestampTime := time.Unix(1000, 0)

	queryId1 := []byte("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce499")
	aggReport1 := oracletypes.Aggregate{
		QueryId:           queryId1,
		AggregateValue:    "10",
		AggregatePower:    uint64(10),
		AggregateReporter: creatorAddr.String(),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId1, uint64(timestampTime.UnixMilli())).Return(aggReport1, nil)
	ok.On("GetTimestampBefore", ctx, queryId1, timestampTime).Return(timestampTime.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", ctx, queryId1, timestampTime).Return(timestampTime.Add(1*time.Hour), nil)
	ok.On("GetCurrentAggregateReport", ctx, queryId1).Return(&aggReport1, timestampTime, nil)
	snapshotKey1 := crypto.Keccak256([]byte(hex.EncodeToString(queryId1) + fmt.Sprint(timestampTime.UnixMilli())))
	snapshot1 := []byte("snapshot-1")
	err = k.AttestSnapshotsByReportMap.Set(ctx, snapshotKey1, types.AttestationSnapshots{
		Snapshots: [][]byte{snapshot1},
	})
	require.NoError(t, err)
	err = k.AttestSnapshotDataMap.Set(ctx, snapshot1, types.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("checkpoint"),
		AttestationTimestamp:   uint64(timestampTime.UnixMilli()),
		PrevReportTimestamp:    uint64(timestampTime.Add(-1 * time.Hour).UnixMilli()),
		NextReportTimestamp:    uint64(0),
		QueryId:                queryId1,
		Timestamp:              uint64(timestampTime.UnixMilli()),
		LastConsensusTimestamp: uint64(timestampTime.Add(-2 * time.Hour).UnixMilli()),
	})
	require.NoError(t, err)

	queryId2 := []byte("f7ac7f444de4e3f6378896b232ce4f97f104725f3d95f1790f6ac8af0e3fcf88")
	aggReport2 := oracletypes.Aggregate{
		QueryId:           queryId2,
		AggregateValue:    "11",
		AggregatePower:    uint64(10),
		AggregateReporter: creatorAddr.String(),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId2, uint64(timestampTime.UnixMilli())).Return(aggReport2, nil)
	ok.On("GetTimestampBefore", ctx, queryId2, timestampTime).Return(timestampTime.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", ctx, queryId2, timestampTime).Return(timestampTime.Add(1*time.Hour), nil)
	ok.On("GetCurrentAggregateReport", ctx, queryId2).Return(&aggReport2, timestampTime, nil)
	snapshotKey2 := crypto.Keccak256([]byte(hex.EncodeToString(queryId2) + fmt.Sprint(timestampTime.UnixMilli())))
	snapshot2 := []byte("snapshot-2")
	err = k.AttestSnapshotsByReportMap.Set(ctx, snapshotKey2, types.AttestationSnapshots{
		Snapshots: [][]byte{snapshot2},
	})
	require.NoError(t, err)
	err = k.AttestSnapshotDataMap.Set(ctx, snapshot2, types.AttestationSnapshotData{
		ValidatorCheckpoint:    []byte("checkpoint"),
		AttestationTimestamp:   uint64(timestampTime.UnixMilli()),
		PrevReportTimestamp:    uint64(timestampTime.Add(-1 * time.Hour).UnixMilli()),
		NextReportTimestamp:    uint64(0),
		QueryId:                queryId2,
		Timestamp:              uint64(timestampTime.UnixMilli()),
		LastConsensusTimestamp: uint64(timestampTime.Add(-2 * time.Hour).UnixMilli()),
	})
	require.NoError(t, err)

	_, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   creatorAddr.String(),
		QueryId:   hex.EncodeToString(queryId1),
		Timestamp: strconv.FormatInt(timestampTime.UnixMilli(), 10),
	})
	require.NoError(t, err)

	_, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   creatorAddr.String(),
		QueryId:   hex.EncodeToString(queryId2),
		Timestamp: strconv.FormatInt(timestampTime.UnixMilli(), 10),
	})
	require.ErrorContains(t, err, "too many external requests")

	requests, err := k.AttestRequestsByHeightMap.Get(ctx, uint64(sdkCtx.BlockHeight()))
	require.NoError(t, err)
	require.Len(t, requests.Requests, 1)
}
