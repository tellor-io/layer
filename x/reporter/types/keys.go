package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "reporter"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_reporter"

	Denom = "loya"

	TipsEscrowPool = "tips_escrow_pool"
)

var (
	ParamsKey                           = []byte("p_reporter")
	ReportersKey                        = collections.NewPrefix(11)
	DelegatorsKey                       = collections.NewPrefix(12)
	TokenOriginsKey                     = collections.NewPrefix(13)
	ReporterAccumulatedCommissionPrefix = collections.NewPrefix(14)
	ReporterOutstandingRewardsPrefix    = collections.NewPrefix(15)
	ReporterCurrentRewardsPrefix        = collections.NewPrefix(16)
	DelegatorStartingInfoPrefix         = collections.NewPrefix(17)
	ReporterHistoricalRewardsPrefix     = collections.NewPrefix(18)
	ReporterDisputeEventPrefix          = collections.NewPrefix(19)
	ReporterDelegatorsIndexPrefix       = collections.NewPrefix(20)
	TokenOriginSnapshotPrefix           = collections.NewPrefix(21)
	DelegatorTipsPrefix                 = collections.NewPrefix(22)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
