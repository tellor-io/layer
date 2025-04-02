package bridge

import (
	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		panic(err)
	}
	if err := k.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: genState.SnapshotLimit}); err != nil {
		panic(err)
	}
	if err := k.BridgeValset.Set(ctx, *genState.BridgeValSet); err != nil {
		panic(err)
	}
	if err := k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{Checkpoint: genState.ValidatorCheckpoint}); err != nil {
		panic(err)
	}
	if err := k.WithdrawalId.Set(ctx, types.WithdrawalId{Id: genState.WithdrawalId}); err != nil {
		panic(err)
	}
	if err := k.LatestCheckpointIdx.Set(ctx, types.CheckpointIdx{Index: genState.LatestValidatorCheckpointIdx}); err != nil {
		panic(err)
	}

	for _, data := range genState.OperatorToEvmAddressMap {
		if err := k.OperatorToEVMAddressMap.Set(ctx, data.OperatorAddress, types.EVMAddress{EVMAddress: data.EvmAddress}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.EvmRegisteredMap {
		if err := k.EVMAddressRegisteredMap.Set(ctx, data.OperatorAddress, types.EVMAddressRegistered{Registered: data.Registered}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.BridgeValsetSigsMap {
		if err := k.BridgeValsetSignaturesMap.Set(ctx, data.Timestamp, types.BridgeValsetSignatures{Signatures: data.ValsetSigs}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.ValidatorCheckpointParamsMap {
		if err := k.ValidatorCheckpointParamsMap.Set(ctx, data.Timestamp, types.ValidatorCheckpointParams{Checkpoint: data.ValidatorCheckpoint, ValsetHash: data.ValidatorSetHash, Timestamp: data.Timestamp, PowerThreshold: data.ValidatorPowerThreshold}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.ValidatorCheckpointIdxMap {
		if err := k.ValidatorCheckpointIdxMap.Set(ctx, data.Index, types.CheckpointTimestamp{Timestamp: data.Timestamp}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.BridgeValsetByTimestampMap {
		if err := k.BridgeValsetByTimestampMap.Set(ctx, data.Timestamp, *data.Valset); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.ValsetTimestampToIdxMap {
		if err := k.ValsetTimestampToIdxMap.Set(ctx, data.Timestamp, types.CheckpointIdx{Index: data.Index}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.AttestSnapshotsByReportMap {
		if err := k.AttestSnapshotsByReportMap.Set(ctx, data.Key, types.AttestationSnapshots{Snapshots: data.Snapshots}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.AttestSnapshotDataMap {
		if err := k.AttestSnapshotDataMap.Set(ctx, data.Key, types.AttestationSnapshotData{
			ValidatorCheckpoint:    data.ValCheckpoint,
			AttestationTimestamp:   data.AttestationTimestamp,
			PrevReportTimestamp:    data.PrevReportTimestamp,
			NextReportTimestamp:    data.NextReportTimestamp,
			QueryId:                data.QueryId,
			Timestamp:              data.Timestamp,
			LastConsensusTimestamp: data.LastConsensusTimestamp,
		}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.SnapshotToAttestationsMap {
		if err := k.SnapshotToAttestationsMap.Set(ctx, data.Snapshot, types.OracleAttestations{Attestations: data.Attestations}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.AttestRequestsByHeightMap {
		requests := make([]*types.AttestationRequest, len(data.Requests))
		for _, snapshot := range data.Requests {
			requests = append(requests, &types.AttestationRequest{Snapshot: snapshot})
		}
		if err := k.AttestRequestsByHeightMap.Set(ctx, data.BlockHeight, types.AttestationRequests{Requests: requests}); err != nil {
			panic(err)
		}
	}

	for _, data := range genState.DepositIdClaimedMap {
		if err := k.DepositIdClaimedMap.Set(ctx, data.DepositId, types.DepositClaimed{Claimed: data.IsClaimed}); err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	var err error
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	snapshotLimit, err := k.SnapshotLimit.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.SnapshotLimit = snapshotLimit.Limit

	bridgeValSet, err := k.BridgeValset.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.BridgeValSet = &bridgeValSet

	validatorCheckpoint, err := k.ValidatorCheckpoint.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.ValidatorCheckpoint = validatorCheckpoint.Checkpoint

	withdrawalId, err := k.WithdrawalId.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.WithdrawalId = withdrawalId.Id

	iterOperaterToEVM, err := k.OperatorToEVMAddressMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	operaterToEVMs := make([]*types.OperatorToEVMAddressMapEntry, 0)
	for ; iterOperaterToEVM.Valid(); iterOperaterToEVM.Next() {
		operatorAddr, err := iterOperaterToEVM.Key()
		if err != nil {
			panic(err)
		}

		evmAddr, err := iterOperaterToEVM.Value()
		if err != nil {
			panic(err)
		}
		operaterToEVMs = append(operaterToEVMs, &types.OperatorToEVMAddressMapEntry{OperatorAddress: operatorAddr, EvmAddress: evmAddr.EVMAddress})
	}
	genesis.OperatorToEvmAddressMap = operaterToEVMs
	err = iterOperaterToEVM.Close()
	if err != nil {
		panic(err)
	}

	iterEVMRegisteredMap, err := k.EVMAddressRegisteredMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	evmRegistered := make([]*types.EVMRegisteredMapEntry, 0)
	for ; iterEVMRegisteredMap.Valid(); iterEVMRegisteredMap.Next() {
		operaterAddr, err := iterEVMRegisteredMap.Key()
		if err != nil {
			panic(err)
		}

		isRegistered, err := iterEVMRegisteredMap.Value()
		if err != nil {
			panic(err)
		}
		evmRegistered = append(evmRegistered, &types.EVMRegisteredMapEntry{OperatorAddress: operaterAddr, Registered: isRegistered.Registered})
	}
	genesis.EvmRegisteredMap = evmRegistered
	err = iterEVMRegisteredMap.Close()
	if err != nil {
		panic(err)
	}

	iterBridgeValSetSigs, err := k.BridgeValsetSignaturesMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	bridgeValsetSigs := make([]*types.BridgeValSetSigsMapEntry, 0)
	for ; iterBridgeValSetSigs.Valid(); iterBridgeValSetSigs.Next() {
		timestamp, err := iterBridgeValSetSigs.Key()
		if err != nil {
			panic(err)
		}

		sigs, err := iterBridgeValSetSigs.Value()
		if err != nil {
			panic(err)
		}
		bridgeValsetSigs = append(bridgeValsetSigs, &types.BridgeValSetSigsMapEntry{Timestamp: timestamp, ValsetSigs: sigs.Signatures})
	}
	genesis.BridgeValsetSigsMap = bridgeValsetSigs
	err = iterBridgeValSetSigs.Close()
	if err != nil {
		panic(err)
	}

	iterValCheckpointParams, err := k.ValidatorCheckpointParamsMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	valCheckpointParamsMap := make([]*types.ValidatorCheckpointParamsStateEntry, 0)
	for ; iterValCheckpointParams.Valid(); iterValCheckpointParams.Next() {
		timestamp, err := iterValCheckpointParams.Key()
		if err != nil {
			panic(err)
		}

		checkpointParams, err := iterValCheckpointParams.Value()
		if err != nil {
			panic(err)
		}
		valCheckpointParamsMap = append(valCheckpointParamsMap, &types.ValidatorCheckpointParamsStateEntry{Timestamp: timestamp, ValidatorTimestamp: checkpointParams.Timestamp, ValidatorPowerThreshold: checkpointParams.PowerThreshold, ValidatorSetHash: checkpointParams.ValsetHash, ValidatorCheckpoint: checkpointParams.Checkpoint})
	}
	genesis.ValidatorCheckpointParamsMap = valCheckpointParamsMap
	err = iterValCheckpointParams.Close()
	if err != nil {
		panic(err)
	}

	iterValCheckpointIdx, err := k.ValidatorCheckpointIdxMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	valCheckpointIdxes := make([]*types.ValidatorCheckpointIdxMapEntry, 0)
	for ; iterValCheckpointIdx.Valid(); iterValCheckpointIdx.Next() {
		idx, err := iterValCheckpointIdx.Key()
		if err != nil {
			panic(err)
		}

		checkpointTimestamp, err := iterValCheckpointIdx.Value()
		if err != nil {
			panic(err)
		}
		valCheckpointIdxes = append(valCheckpointIdxes, &types.ValidatorCheckpointIdxMapEntry{Index: idx, Timestamp: checkpointTimestamp.Timestamp})
	}
	genesis.ValidatorCheckpointIdxMap = valCheckpointIdxes
	err = iterValCheckpointIdx.Close()
	if err != nil {
		panic(err)
	}

	latestCheckpointIdx, err := k.LatestCheckpointIdx.Get(ctx)
	if err != nil {
		panic(err)
	}
	genesis.LatestValidatorCheckpointIdx = latestCheckpointIdx.Index

	iterBridgeValsetByTimestamp, err := k.BridgeValsetByTimestampMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	bridgeValsetByTimestamps := make([]*types.BridgeValsetByTimestampMapEntry, 0)
	for ; iterBridgeValsetByTimestamp.Valid(); iterBridgeValsetByTimestamp.Next() {
		timestamp, err := iterBridgeValsetByTimestamp.Key()
		if err != nil {
			panic(err)
		}

		valset, err := iterBridgeValsetByTimestamp.Value()
		if err != nil {
			panic(err)
		}
		bridgeValsetByTimestamps = append(bridgeValsetByTimestamps, &types.BridgeValsetByTimestampMapEntry{Timestamp: timestamp, Valset: &valset})
	}
	genesis.BridgeValsetByTimestampMap = bridgeValsetByTimestamps
	err = iterBridgeValsetByTimestamp.Close()
	if err != nil {
		panic(err)
	}

	iterValsetTimestampToIdx, err := k.ValsetTimestampToIdxMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	valsetTimestampToIdxes := make([]*types.ValsetTimestampToIdxMapEntry, 0)
	for ; iterValsetTimestampToIdx.Valid(); iterValsetTimestampToIdx.Next() {
		timestamp, err := iterValsetTimestampToIdx.Key()
		if err != nil {
			panic(err)
		}

		idx, err := iterValsetTimestampToIdx.Value()
		if err != nil {
			panic(err)
		}
		valsetTimestampToIdxes = append(valsetTimestampToIdxes, &types.ValsetTimestampToIdxMapEntry{Timestamp: timestamp, Index: idx.Index})
	}
	genesis.ValsetTimestampToIdxMap = valsetTimestampToIdxes
	err = iterValsetTimestampToIdx.Close()
	if err != nil {
		panic(err)
	}

	iterAttestSnapshotsByReport, err := k.AttestSnapshotsByReportMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	attestSnapshotsbyReports := make([]*types.AttestSnapshotsByReportMapEntry, 0)
	for ; iterAttestSnapshotsByReport.Valid(); iterAttestSnapshotsByReport.Next() {
		key, err := iterAttestSnapshotsByReport.Key()
		if err != nil {
			panic(err)
		}

		attestionSnapshot, err := iterAttestSnapshotsByReport.Value()
		if err != nil {
			panic(err)
		}
		attestSnapshotsbyReports = append(attestSnapshotsbyReports, &types.AttestSnapshotsByReportMapEntry{Key: key, Snapshots: attestionSnapshot.Snapshots})
	}
	genesis.AttestSnapshotsByReportMap = attestSnapshotsbyReports
	err = iterAttestSnapshotsByReport.Close()
	if err != nil {
		panic(err)
	}

	iterAttestSnapshotData, err := k.AttestSnapshotDataMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	attestSnapshotDataEntries := make([]*types.AttestSnapshotDataMapEntry, 0)
	for ; iterAttestSnapshotData.Valid(); iterAttestSnapshotData.Next() {
		key, err := iterAttestSnapshotData.Key()
		if err != nil {
			panic(err)
		}

		attestSnapshotData, err := iterAttestSnapshotData.Value()
		if err != nil {
			panic(err)
		}
		attestSnapshotDataEntries = append(attestSnapshotDataEntries, &types.AttestSnapshotDataMapEntry{Key: key, ValCheckpoint: attestSnapshotData.ValidatorCheckpoint, PrevReportTimestamp: attestSnapshotData.PrevReportTimestamp, NextReportTimestamp: attestSnapshotData.NextReportTimestamp, QueryId: attestSnapshotData.QueryId, Timestamp: attestSnapshotData.Timestamp, LastConsensusTimestamp: attestSnapshotData.LastConsensusTimestamp, AttestationTimestamp: attestSnapshotData.AttestationTimestamp})
	}
	genesis.AttestSnapshotDataMap = attestSnapshotDataEntries
	err = iterAttestSnapshotData.Close()
	if err != nil {
		panic(err)
	}

	iterSnapshotToAttestations, err := k.SnapshotToAttestationsMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	snapshotToAttestations := make([]*types.SnapshotToAttestationsMapEntry, 0)
	for ; iterSnapshotToAttestations.Valid(); iterSnapshotToAttestations.Next() {
		snapshot, err := iterSnapshotToAttestations.Key()
		if err != nil {
			panic(err)
		}

		attestations, err := iterSnapshotToAttestations.Value()
		if err != nil {
			panic(err)
		}
		snapshotToAttestations = append(snapshotToAttestations, &types.SnapshotToAttestationsMapEntry{Snapshot: snapshot, Attestations: attestations.Attestations})
	}
	genesis.SnapshotToAttestationsMap = snapshotToAttestations
	err = iterSnapshotToAttestations.Close()
	if err != nil {
		panic(err)
	}

	iterAttestRequestsByHeight, err := k.AttestRequestsByHeightMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	attestRequestsByHeight := make([]*types.AttestRequestsByHeightMapEntry, 0)
	for ; iterAttestRequestsByHeight.Valid(); iterAttestRequestsByHeight.Next() {
		block_height, err := iterAttestRequestsByHeight.Key()
		if err != nil {
			panic(err)
		}

		requests, err := iterAttestRequestsByHeight.Value()
		if err != nil {
			panic(err)
		}
		attestRequests := make([][]byte, 0)
		for _, data := range requests.Requests {
			attestRequests = append(attestRequests, data.Snapshot)
		}
		attestRequestsByHeight = append(attestRequestsByHeight, &types.AttestRequestsByHeightMapEntry{BlockHeight: block_height, Requests: attestRequests})
	}
	genesis.AttestRequestsByHeightMap = attestRequestsByHeight
	err = iterAttestRequestsByHeight.Close()
	if err != nil {
		panic(err)
	}

	iterDepositIdClaimed, err := k.DepositIdClaimedMap.IterateRaw(ctx, nil, nil, collections.OrderDescending)
	if err != nil {
		panic(err)
	}
	depositsClaimed := make([]*types.DepositIdClaimedMapEntry, 0)
	for ; iterDepositIdClaimed.Valid(); iterDepositIdClaimed.Next() {
		deposit_id, err := iterDepositIdClaimed.Key()
		if err != nil {
			panic(err)
		}

		isClaimed, err := iterDepositIdClaimed.Value()
		if err != nil {
			panic(err)
		}
		depositsClaimed = append(depositsClaimed, &types.DepositIdClaimedMapEntry{DepositId: deposit_id, IsClaimed: isClaimed.Claimed})
	}
	genesis.DepositIdClaimedMap = depositsClaimed
	err = iterDepositIdClaimed.Close()
	if err != nil {
		panic(err)
	}
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
