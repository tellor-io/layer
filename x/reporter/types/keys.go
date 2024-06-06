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
	ParamsKey                       = []byte("p_reporter")
	ReportersKey                    = collections.NewPrefix(11)
	DelegatorsKey                   = collections.NewPrefix(12)
	ReporterDelegatorsIndexPrefix   = collections.NewPrefix(13)
	DelegatorTipsPrefix             = collections.NewPrefix(14)
	DisputedDelegationAmountsPrefix = collections.NewPrefix(15)
	FeePaidFromStakePrefix          = collections.NewPrefix(16)
	StakeTrackerPrefix              = collections.NewPrefix(17)
	ReporterPrefix                  = collections.NewPrefix(18)
	TempPrefix                      = collections.NewPrefix(19)
	DelegatorAmountPrefix           = collections.NewPrefix(20)
)
