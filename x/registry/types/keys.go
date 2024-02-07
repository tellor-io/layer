package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "registry"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_registry"
)

var (
	// ParamsKey
	ParamsKey = collections.NewPrefix(11)

	// RegistryKey
	QueryRegistryKey = collections.NewPrefix(12)

	// SpecRegistryKey
	SpecRegistryKey = collections.NewPrefix(13)
)
