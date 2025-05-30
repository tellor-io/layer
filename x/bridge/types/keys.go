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
	ParamsKey                                  = collections.NewPrefix(0)  // Prefix for params key
	BridgeValsetKey                            = collections.NewPrefix(1)  // Prefix for bridge_valset key
	ValidatorCheckpointKey                     = collections.NewPrefix(2)  // Prefix for validator_checkpoint key
	OperatorToEVMAddressMapKey                 = collections.NewPrefix(3)  // Prefix for operator_to_evm_address_map key
	EVMToOperatorAddressMapKey                 = collections.NewPrefix(4)  // Prefix for evm_to_operator_address_map key
	EVMAddressRegisteredMapKey                 = collections.NewPrefix(5)  // Prefix for evm_address_registered_map key
	BridgeValsetSignaturesMapKey               = collections.NewPrefix(6)  // Prefix for bridge_valset_signatures_map key
	ValidatorCheckpointParamsMapKey            = collections.NewPrefix(7)  // Prefix for validator_checkpoint_params key
	ValidatorCheckpointIdxMapKey               = collections.NewPrefix(8)  // Prefix for validator_checkpoint_idx_map key
	LatestCheckpointIdxKey                     = collections.NewPrefix(9)  // Prefix for latest_checkpoint_idx key
	OracleAttestationsMapKey                   = collections.NewPrefix(10) // Prefix for oracle_attestations_map key
	BridgeValsetByTimestampMapKey              = collections.NewPrefix(11) // Prefix for bridge_valset_by_timestamp_map key
	ValsetTimestampToIdxMapKey                 = collections.NewPrefix(12) // Prefix for valset_timestamp_to_idx_map key
	AttestSnapshotsByReportMapKey              = collections.NewPrefix(13) // Prefix for attest_snapshots_by_report_map key
	AttestSnapshotDataMapKey                   = collections.NewPrefix(14) // Prefix for attest_snapshot_data_map key
	SnapshotToAttestationsMapKey               = collections.NewPrefix(15) // Prefix for snapshot_to_attestations_map key
	AttestRequestsByHeightMapKey               = collections.NewPrefix(16) // Prefix for attest_requests_by_height_map key
	WithdrawalIdKey                            = collections.NewPrefix(17) // Prefix for withdrawal_id key
	DepositIdClaimedMapKey                     = collections.NewPrefix(18) // Prefix for deposit_id_claimed_map key
	SnapshotLimitKey                           = collections.NewPrefix(19) // Prefix for snapshot_limit key
	AttestationEvidenceSubmittedKey            = collections.NewPrefix(20) // Prefix for attestation_evidence_submitted_map key
	ValidatorCheckpointByCheckpointIndexPrefix = collections.NewPrefix(21) // Prefix for validator_checkpoint_by_checkpoint_index
)
