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
	SelectorsKey                    = collections.NewPrefix(12)
	ReporterSelectorsIndexPrefix    = collections.NewPrefix(13)
	SelectorTipsPrefix              = collections.NewPrefix(14)
	DisputedDelegationAmountsPrefix = collections.NewPrefix(15)
	FeePaidFromStakePrefix          = collections.NewPrefix(16)
	StakeTrackerPrefix              = collections.NewPrefix(17)
	ReporterPrefix                  = collections.NewPrefix(18)
	TipPrefix                       = collections.NewPrefix(19)
	TbrPrefix                       = collections.NewPrefix(20)
	ClaimStatusPrefix               = collections.NewPrefix(21)
	ReporterPeriodDataPrefix        = collections.NewPrefix(22)
	DistributionQueuePrefix         = collections.NewPrefix(23)
	DistributionQueueCounterPrefix  = collections.NewPrefix(24)
	LastValSetUpdateHeightPrefix    = collections.NewPrefix(25)
	StakeRecalcFlagPrefix           = collections.NewPrefix(26)
	RecalcAtTimePrefix              = collections.NewPrefix(27)
)
