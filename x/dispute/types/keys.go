package types

import (
	"encoding/binary"
	"fmt"
)

const (
	// ModuleName defines the module name
	ModuleName = "dispute"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_dispute"

	// DisputesKey defines the disputes key
	DisputesKey = "disputes"

	// DisputeCountKey defines the dispute count key
	DisputeCountKey = "dispute-count"

	// OpenDisputeIdsKey defines the open dispute ids key
	OpenDisputeIdsKey = "open-dispute-ids"

	// VotesKey defines the votes key
	VotesKey = "votes"

	// Voter Power key
	VoterPowerKey = "voter-power"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

func DisputesKeyPrefix() []byte {
	return KeyPrefix(DisputesKey)
}

func DisputeIdBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

func OpenDisputeIdsKeyPrefix() []byte {
	return KeyPrefix(OpenDisputeIdsKey)
}

func VotesKeyPrefix() []byte {
	return KeyPrefix(VotesKey)
}

func VoterKeyPrefix(voter string, id uint64) []byte {
	return KeyPrefix(fmt.Sprintf("%s:%d", voter, id))
}

func TallyKeyPrefix(id uint64) []byte {
	return KeyPrefix(fmt.Sprintf("%s:%d", "VoteTally", id))
}

func VoterPowerKeyPrefix() []byte {
	return KeyPrefix(VoterPowerKey)
}
