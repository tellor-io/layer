package bridge_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/nullify"
	"github.com/tellor-io/layer/x/bridge"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:                       types.DefaultParams(),
		SnapshotLimit:                1000,
		BridgeValSet:                 nil,
		ValidatorCheckpoint:          nil,
		WithdrawalId:                 0,
		OperatorToEvmAddressMap:      make([]*types.OperatorToEVMAddressMapEntry, 0),
		EvmRegisteredMap:             make([]*types.EVMRegisteredMapEntry, 0),
		BridgeValsetSigsMap:          make([]*types.BridgeValSetSigsMapEntry, 0),
		ValidatorCheckpointParamsMap: make([]*types.ValidatorCheckpointParamsStateEntry, 0),
		ValidatorCheckpointIdxMap:    make([]*types.ValidatorCheckpointIdxMapEntry, 0),
		LatestValidatorCheckpointIdx: 0,
		BridgeValsetByTimestampMap:   make([]*types.BridgeValsetByTimestampMapEntry, 0),
		ValsetTimestampToIdxMap:      make([]*types.ValsetTimestampToIdxMapEntry, 0),
		DepositIdClaimedMap:          make([]*types.DepositIdClaimedMapEntry, 0),
	}

	k, _, _, _, _, _, _, ctx := keepertest.BridgeKeeper(t)
	require.NotPanics(t, func() { bridge.InitGenesis(ctx, k, genesisState) })
	got := bridge.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	genesisState.WithdrawalId = 5
	genesisState.BridgeValSet = &types.BridgeValidatorSet{BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("test address"), Power: 1000}}}
	genesisState.ValidatorCheckpoint = []byte("checkpoint")
	genesisState.OperatorToEvmAddressMap = []*types.OperatorToEVMAddressMapEntry{{OperatorAddress: "operating", EvmAddress: []byte("evm")}}
	genesisState.EvmRegisteredMap = []*types.EVMRegisteredMapEntry{{OperatorAddress: "test", Registered: true}}
	genesisState.BridgeValsetSigsMap = []*types.BridgeValSetSigsMapEntry{{Timestamp: 1000000, ValsetSigs: [][]byte{[]byte("sig1"), []byte("sig2")}}}
	genesisState.ValidatorCheckpointParamsMap = []*types.ValidatorCheckpointParamsStateEntry{{Timestamp: 1000000, ValidatorTimestamp: 100, ValidatorPowerThreshold: 10000, ValidatorSetHash: []byte("valset"), ValidatorCheckpoint: []byte("checkpoint")}}
	genesisState.ValidatorCheckpointIdxMap = []*types.ValidatorCheckpointIdxMapEntry{{Index: 10, Timestamp: 5000}}
	genesisState.LatestValidatorCheckpointIdx = 10
	genesisState.BridgeValsetByTimestampMap = []*types.BridgeValsetByTimestampMapEntry{{Timestamp: 10, Valset: &types.BridgeValidatorSet{BridgeValidatorSet: []*types.BridgeValidator{{EthereumAddress: []byte("test address"), Power: 1000}}}}}
	genesisState.ValsetTimestampToIdxMap = []*types.ValsetTimestampToIdxMapEntry{{Timestamp: 1000, Index: 6}}
	genesisState.DepositIdClaimedMap = []*types.DepositIdClaimedMapEntry{{DepositId: 1, IsClaimed: true}, {DepositId: 2, IsClaimed: false}}

	k, _, _, _, _, _, _, ctx = keepertest.BridgeKeeper(t)
	ctx = ctx.WithBlockHeight(10)
	require.NotPanics(t, func() { bridge.InitGenesis(ctx, k, genesisState) })
	got = bridge.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.Equal(t, genesisState.WithdrawalId, got.WithdrawalId)
	require.Equal(t, genesisState.BridgeValSet, got.BridgeValSet)
	require.Equal(t, genesisState.ValidatorCheckpoint, got.ValidatorCheckpoint)
	require.Equal(t, genesisState.EvmRegisteredMap, got.EvmRegisteredMap)
	require.Equal(t, genesisState.BridgeValsetSigsMap, got.BridgeValsetSigsMap)
	require.Equal(t, genesisState.ValidatorCheckpointParamsMap, got.ValidatorCheckpointParamsMap)
	require.Equal(t, genesisState.ValidatorCheckpointIdxMap, got.ValidatorCheckpointIdxMap)
	require.Equal(t, genesisState.LatestValidatorCheckpointIdx, got.LatestValidatorCheckpointIdx)
	require.Equal(t, genesisState.BridgeValsetByTimestampMap, got.BridgeValsetByTimestampMap)
	require.Equal(t, genesisState.ValsetTimestampToIdxMap, got.ValsetTimestampToIdxMap)
	require.Equal(t, genesisState.DepositIdClaimedMap, got.DepositIdClaimedMap)

	// this line is used by starport scaffolding # genesis/test/assert
}
