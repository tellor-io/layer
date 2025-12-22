package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "oracle"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_oracle"

	// ParamsKey
	ParamsKey = "oracle_params"
)

var (
	CommitsPrefix   = collections.NewPrefix(0)
	TipsPrefix      = collections.NewPrefix(1)
	TipsIndexPrefix = collections.NewPrefix(2)

	ReportsPrefix              = collections.NewPrefix(3)
	ReportsHeightIndexPrefix   = collections.NewPrefix(4)
	ReportsReporterIndexPrefix = collections.NewPrefix(5)

	AggregatesPrefix = collections.NewPrefix(6)
	NoncesPrefix     = collections.NewPrefix(7)
	TotalTipsPrefix  = collections.NewPrefix(8)

	QuerySeqPrefix                   = collections.NewPrefix(9)
	QueryTipPrefix                   = collections.NewPrefix(10)
	ReportsIdIndexPrefix             = collections.NewPrefix(11)
	QueryCyclePrefix                 = collections.NewPrefix(12)
	CycleSeqPrefix                   = collections.NewPrefix(13)
	NextInListPrefix                 = collections.NewPrefix(15)
	QueryRevealedIdsIndexPrefix      = collections.NewPrefix(16)
	CyclelistPrefix                  = collections.NewPrefix(17)
	QueryTypeIndexPrefix             = collections.NewPrefix(18)
	AggregatesHeightIndexPrefix      = collections.NewPrefix(19)
	TipsBlockIndexPrefix             = collections.NewPrefix(20)
	TipperTotalPrefix                = collections.NewPrefix(21)
	AggregatesMicroHeightIndexPrefix = collections.NewPrefix(22)
	QueryDataLimitPrefix             = collections.NewPrefix(23)
	ValuesWeightSumPrefix            = collections.NewPrefix(25)
	ValuesPrefix                     = collections.NewPrefix(26)
	AggregateValuePrefix             = collections.NewPrefix(27)
	ValuesPowerPrefix                = collections.NewPrefix(28)
	QueryByExpirationPrefix          = collections.NewPrefix(29)
	BridgeDepositQueuePrefix         = collections.NewPrefix(30)
	NoStakeReportsPrefix             = collections.NewPrefix(31)
	ReporterIndexPrefix              = collections.NewPrefix(32)
	NoStakeReportedQueriesPrefix     = collections.NewPrefix(33)
	MaxBatchSizePrefix               = collections.NewPrefix(34)

	// Liveness reward storage prefixes
	LivenessRecordsPrefix          = collections.NewPrefix(35)
	CycleCountPrefix               = collections.NewPrefix(36)
	DustPrefix                     = collections.NewPrefix(37)
	QueryOpportunitiesPrefix       = collections.NewPrefix(38)
	ReporterQueryShareSumPrefix    = collections.NewPrefix(39)
	ReporterStandardShareSumPrefix = collections.NewPrefix(40)
	NonStandardQueriesPrefix       = collections.NewPrefix(41)
	StandardOpportunitiesPrefix    = collections.NewPrefix(42)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func ParamsKeyPrefix() []byte {
	return KeyPrefix(ParamsKey)
}
