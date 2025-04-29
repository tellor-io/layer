package types

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

type DisputesIndex struct {
	DisputeByReporter *indexes.Multi[[]byte, uint64, Dispute]
	OpenDisputes      *indexes.Multi[bool, uint64, Dispute]
	PendingExecution  *indexes.Multi[bool, uint64, Dispute] // New index for PendingExecution
}

func (a DisputesIndex) IndexesList() []collections.Index[uint64, Dispute] {
	return []collections.Index[uint64, Dispute]{a.DisputeByReporter, a.OpenDisputes, a.PendingExecution}
}

func NewDisputesIndex(sb *collections.SchemaBuilder) DisputesIndex {
	return DisputesIndex{
		DisputeByReporter: indexes.NewMulti(
			sb, DisputesByReporterIndexPrefix, "dispute_by_reporter",
			collections.BytesKey, collections.Uint64Key,
			func(k uint64, dispute Dispute) ([]byte, error) {
				reporterKey := fmt.Sprintf("%s:%x", dispute.InitialEvidence.Reporter, dispute.HashId)
				return []byte(reporterKey), nil
			},
		),
		OpenDisputes: indexes.NewMulti(
			sb, OpenDisputesIndexPrefix, "open_disputes",
			collections.BoolKey, collections.Uint64Key,
			func(k uint64, dispute Dispute) (bool, error) {
				return dispute.Open, nil
			},
		),
		PendingExecution: indexes.NewMulti(
			sb, PendingExecutionIndexPrefix, "pending_execution",
			collections.BoolKey, collections.Uint64Key,
			func(k uint64, dispute Dispute) (bool, error) {
				return dispute.PendingExecution, nil
			},
		),
	}
}

type VotersVoteIndex struct {
	VotersById *indexes.Multi[uint64, collections.Pair[uint64, []byte], Voter]
}

func (a VotersVoteIndex) IndexesList() []collections.Index[collections.Pair[uint64, []byte], Voter] {
	return []collections.Index[collections.Pair[uint64, []byte], Voter]{a.VotersById}
}

func NewVotersIndex(sb *collections.SchemaBuilder) VotersVoteIndex {
	return VotersVoteIndex{
		VotersById: indexes.NewMulti(
			sb, VotersByIdIndexPrefix, "voters_by_id",
			collections.Uint64Key, collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey),
			func(k collections.Pair[uint64, []byte], _ Voter) (uint64, error) {
				return k.K1(), nil
			},
		),
	}
}
