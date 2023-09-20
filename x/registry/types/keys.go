package types

const (
	// ModuleName defines the module name
	ModuleName = "registry"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_registry"

	// RegistryKey
	QueryRegistryKey = "query_registry_key"

	// SpecRegistryKey
	SpecRegistryKey = "spec_registry_key"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
