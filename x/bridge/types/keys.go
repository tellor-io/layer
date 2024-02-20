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
	ParamsKey                       = collections.NewPrefix(0) // Prefix for params key
	BridgeValsetKey                 = collections.NewPrefix(1) // Prefix for bridge_valset key
	ValidatorCheckpointKey          = collections.NewPrefix(2) // Prefix for validator_checkpoint key
	OperatorToEVMAddressMapKey      = collections.NewPrefix(3) // Prefix for operator_to_evm_address_map key
	BridgeValsetSignaturesMapKey    = collections.NewPrefix(4) // Prefix for bridge_valset_signatures_map key
	ValidatorCheckpointParamsMapKey = collections.NewPrefix(5) // Prefix for validator_checkpoint_params key
	ValidatorCheckpointIdxMapKey    = collections.NewPrefix(6) // Prefix for validator_checkpoint_idx_map key
	LatestCheckpointIdxKey          = collections.NewPrefix(7) // Prefix for latest_checkpoint_idx key
)
