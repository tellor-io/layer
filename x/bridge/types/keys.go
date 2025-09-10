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
	ParamsKey                                  = collections.NewPrefix(0)  // params key
	BridgeValsetKey                            = collections.NewPrefix(1)  // bridge_valset key
	ValidatorCheckpointKey                     = collections.NewPrefix(2)  // validator_checkpoint key
	OperatorToEVMAddressMapKey                 = collections.NewPrefix(3)  // operator_to_evm_address_map key
	EVMAddressRegisteredMapKey                 = collections.NewPrefix(4)  // evm_address_registered_map key
	BridgeValsetSignaturesMapKey               = collections.NewPrefix(5)  // bridge_valset_signatures_map key
	ValidatorCheckpointParamsMapKey            = collections.NewPrefix(6)  // validator_checkpoint_params key
	ValidatorCheckpointIdxMapKey               = collections.NewPrefix(7)  // validator_checkpoint_idx_map key
	LatestCheckpointIdxKey                     = collections.NewPrefix(8)  // latest_checkpoint_idx key
	OracleAttestationsMapKey                   = collections.NewPrefix(9)  // oracle_attestations_map key
	BridgeValsetByTimestampMapKey              = collections.NewPrefix(10) // bridge_valset_by_timestamp_map key
	ValsetTimestampToIdxMapKey                 = collections.NewPrefix(11) // valset_timestamp_to_idx_map key
	AttestSnapshotsByReportMapKey              = collections.NewPrefix(12) // attest_snapshots_by_report_map key
	AttestSnapshotDataMapKey                   = collections.NewPrefix(13) // attest_snapshot_data_map key
	SnapshotToAttestationsMapKey               = collections.NewPrefix(14) // snapshot_to_attestations_map key
	AttestRequestsByHeightMapKey               = collections.NewPrefix(15) // attest_requests_by_height_map key
	WithdrawalIdKey                            = collections.NewPrefix(16) // withdrawal_id key
	DepositIdClaimedMapKey                     = collections.NewPrefix(17) // deposit_id_claimed_map key
	SnapshotLimitKey                           = collections.NewPrefix(18) // snapshot_limit key
	EVMToOperatorAddressMapKey                 = collections.NewPrefix(19) // evm_to_operator_address_map key
	AttestationEvidenceSubmittedKey            = collections.NewPrefix(20) // attestation_evidence_submitted_map key
	ValidatorCheckpointByCheckpointIndexPrefix = collections.NewPrefix(21) // validator_checkpoint_by_checkpoint_index
	ValsetSignatureEvidenceSubmittedKey        = collections.NewPrefix(22) // valset_signature_evidence_submitted_map
	CrossNetworkAddressMapKey                  = collections.NewPrefix(23) // cross_network_address_map key
)
