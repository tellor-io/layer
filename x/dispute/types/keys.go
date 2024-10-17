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
	DisputesPrefix                           = collections.NewPrefix(1)
	DisputesByReporterIndexPrefix            = collections.NewPrefix(2)
	DisputesCountIndexPrefix                 = collections.NewPrefix(3)
	VotesPrefix                              = collections.NewPrefix(5)
	VoterVotePrefix                          = collections.NewPrefix(6)
	VotersByIdIndexPrefix                    = collections.NewPrefix(7)
	UserPowerIndexPrefix                     = collections.NewPrefix(8)
	ReporterPowerIndexPrefix                 = collections.NewPrefix(9)
	ReportersWithDelegatorsVotedBeforePrefix = collections.NewPrefix(10)
	TeamVoterPrefix                          = collections.NewPrefix(11)
	UsersGroupPrefix                         = collections.NewPrefix(12)
	BlockInfoPrefix                          = collections.NewPrefix(13)
	OpenDisputesIndexPrefix                  = collections.NewPrefix(14)
	DisputeFeePayerPrefix                    = collections.NewPrefix(15)
	DustKeyPrefix                            = collections.NewPrefix(16)
	VoteCountsByGroupPrefix                  = collections.NewPrefix(17)
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func ParamsKeyPrefix() []byte {
	return KeyPrefix(ParamsKey)
}
