package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "dispute"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// ParamsKey defines the primary module store key
	ParamsKey = "params"
)

var (
	DisputesPrefix                = collections.NewPrefix(1)
	DisputesByReporterIndexPrefix = collections.NewPrefix(2)
	DisputesCountIndexPrefix      = collections.NewPrefix(3)
	OpenDisputeIdsPrefix          = collections.NewPrefix(4)
	VotesPrefix                   = collections.NewPrefix(5)
	VoterVotePrefix               = collections.NewPrefix(6)
	VotersByIdIndexPrefix         = collections.NewPrefix(7)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
func ParamsKeyPrefix() []byte {
	return KeyPrefix(ParamsKey)
}
