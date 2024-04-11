package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "bridge"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_bridge"
)

var (
	ParamsKey                       = collections.NewPrefix(0)  // Prefix for params key
	BridgeValsetKey                 = collections.NewPrefix(1)  // Prefix for bridge_valset key
	ValidatorCheckpointKey          = collections.NewPrefix(2)  // Prefix for validator_checkpoint key
	OperatorToEVMAddressMapKey      = collections.NewPrefix(3)  // Prefix for operator_to_evm_address_map key
	BridgeValsetSignaturesMapKey    = collections.NewPrefix(4)  // Prefix for bridge_valset_signatures_map key
	ValidatorCheckpointParamsMapKey = collections.NewPrefix(5)  // Prefix for validator_checkpoint_params key
	ValidatorCheckpointIdxMapKey    = collections.NewPrefix(6)  // Prefix for validator_checkpoint_idx_map key
	LatestCheckpointIdxKey          = collections.NewPrefix(7)  // Prefix for latest_checkpoint_idx key
	OracleAttestationsMapKey        = collections.NewPrefix(8)  // Prefix for oracle_attestations_map key
	BridgeValsetByTimestampMapKey   = collections.NewPrefix(9)  // Prefix for bridge_valset_by_timestamp_map key
	ValsetTimestampToIdxMapKey      = collections.NewPrefix(10) // Prefix for valset_timestamp_to_idx_map key
	AttestSnapshotsByReportMapKey   = collections.NewPrefix(11) // Prefix for attest_snapshots_by_report_map key
	AttestSnapshotDataMapKey        = collections.NewPrefix(12) // Prefix for attest_snapshot_data_map key
	SnapshotToAttestationsMapKey    = collections.NewPrefix(13) // Prefix for snapshot_to_attestations_map key
	AttestRequestsByHeightMapKey    = collections.NewPrefix(14) // Prefix for attest_requests_by_height_map key
	WithdrawalIdKey                 = collections.NewPrefix(15) // Prefix for withdrawal_id key
)
