package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "reporter"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_reporter"
)

var (
	ParamsKey       = []byte("p_reporter")
	ReportersKey    = collections.NewPrefix(11)
	DelegatorsKey   = collections.NewPrefix(12)
	TokenOriginsKey = collections.NewPrefix(13)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
