package fork

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StreamModuleStateData represents a single item from each array in ModuleStateData
type StreamModuleStateData struct {
	Disputes                        []*disputetypes.DisputeStateEntry                         `json:"disputes,omitempty"`
	Votes                           []*disputetypes.VotesStateEntry                           `json:"votes,omitempty"`
	Voters                          []*disputetypes.VoterStateEntry                           `json:"voters,omitempty"`
	ReportersWithDelegatorsWhoVoted []*disputetypes.ReportersWithDelegatorsWhoVotedStateEntry `json:"reporters_with_delegators_who_voted,omitempty"`
	BlockInfo                       []*disputetypes.BlockInfoStateEntry                       `json:"block_info,omitempty"`
	DisputeFeePayer                 []*disputetypes.DisputeFeePayerStateEntry                 `json:"dispute_fee_payer,omitempty"`
	Dust                            *math.Int                                                 `json:"dust,omitempty"`
}

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec, pathToFile string) error {
	path := filepath.Join(
		pathToFile,
		"dispute_module_state.json",
	)

	// Open the JSON file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a decoder that will decode our JSON objects
	decoder := json.NewDecoder(file)

	// Read opening bracket of the JSON file
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Process disputes array
	if err := processDisputesSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process votes array
	if err := processVotesSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process voters array
	if err := processVotersSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process reporters with delegators array
	if err := processReportersWithDelegatorsSection(ctx, decoder, storeService); err != nil {
		return err
	}

	// Process block info array
	if err := processBlockInfoSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process dispute fee payer array
	if err := processDisputeFeePayerSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process dust value
	if err := processDustSection(ctx, decoder, storeService); err != nil {
		return err
	}

	// Process vote counts by group array (if present)
	if err := processVoteCountsSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	return nil
}

type DisputesIndex struct {
	DisputeByReporter *indexes.Multi[[]byte, uint64, disputetypes.Dispute]
	OpenDisputes      *indexes.Multi[bool, uint64, disputetypes.Dispute]
	PendingExecution  *indexes.Multi[bool, uint64, disputetypes.Dispute] // New index for PendingExecution
}

func (a DisputesIndex) IndexesList() []collections.Index[uint64, disputetypes.Dispute] {
	return []collections.Index[uint64, disputetypes.Dispute]{a.DisputeByReporter, a.OpenDisputes, a.PendingExecution}
}

func NewDisputesIndex(sb *collections.SchemaBuilder) DisputesIndex {
	return DisputesIndex{
		DisputeByReporter: indexes.NewMulti(
			sb, disputetypes.DisputesByReporterIndexPrefix, "dispute_by_reporter",
			collections.BytesKey, collections.Uint64Key,
			func(k uint64, dispute disputetypes.Dispute) ([]byte, error) {
				reporterKey := fmt.Sprintf("%s:%x", dispute.InitialEvidence.Reporter, dispute.HashId)
				return []byte(reporterKey), nil
			},
		),
		OpenDisputes: indexes.NewMulti(
			sb, disputetypes.OpenDisputesIndexPrefix, "open_disputes",
			collections.BoolKey, collections.Uint64Key,
			func(k uint64, dispute disputetypes.Dispute) (bool, error) {
				return dispute.Open, nil
			},
		),
		PendingExecution: indexes.NewMulti(
			sb, disputetypes.PendingExecutionIndexPrefix, "pending_execution",
			collections.BoolKey, collections.Uint64Key,
			func(k uint64, dispute disputetypes.Dispute) (bool, error) {
				return dispute.PendingExecution, nil
			},
		),
	}
}

func processDisputesSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "disputes" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "disputes" {
		return fmt.Errorf("expected disputes section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	disputesIndexMap := collections.NewIndexedMap(sb,
		disputetypes.DisputesPrefix,
		"disputes",
		collections.Uint64Key,
		codec.CollValue[disputetypes.Dispute](cdc),
		NewDisputesIndex(sb),
	)
	// Read opening bracket of disputes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry disputetypes.DisputeStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		if err := disputesIndexMap.Set(ctx, entry.DisputeId, *entry.Dispute); err != nil {
			return err
		}
	}

	// Read closing bracket of disputes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processVotesSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "votes" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "votes" {
		return fmt.Errorf("expected votes section, got %v", t)
	}

	// votesStore := prefix.NewStore(store, disputetypes.VotesPrefix)

	sb := collections.NewSchemaBuilder(storeService)
	votesMap := collections.NewMap(sb,
		disputetypes.VotesPrefix,
		"votes",
		collections.Uint64Key,
		codec.CollValue[disputetypes.Vote](cdc),
	)
	// Read opening bracket of votes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry disputetypes.VotesStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		if err := votesMap.Set(ctx, entry.DisputeId, *entry.Vote); err != nil {
			return err
		}
	}

	// Read closing bracket of votes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

type VotersVoteIndex struct {
	VotersById *indexes.Multi[uint64, collections.Pair[uint64, []byte], disputetypes.Voter]
}

func (a VotersVoteIndex) IndexesList() []collections.Index[collections.Pair[uint64, []byte], disputetypes.Voter] {
	return []collections.Index[collections.Pair[uint64, []byte], disputetypes.Voter]{a.VotersById}
}

func NewVotersIndex(sb *collections.SchemaBuilder) VotersVoteIndex {
	return VotersVoteIndex{
		VotersById: indexes.NewMulti(
			sb, disputetypes.VotersByIdIndexPrefix, "voters_by_id",
			collections.Uint64Key, collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey),
			func(k collections.Pair[uint64, []byte], _ disputetypes.Voter) (uint64, error) {
				return k.K1(), nil
			},
		),
	}
}

func processVotersSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "voters" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "voters" {
		return fmt.Errorf("expected voters section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	votersMap := collections.NewIndexedMap(sb,
		disputetypes.VoterVotePrefix,
		"voter_vote",
		collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey),
		codec.CollValue[disputetypes.Voter](cdc),
		NewVotersIndex(sb),
	)

	// Read opening bracket of voters array
	if _, err := decoder.Token(); err != nil {
		return err
	}
	// Read array values
	for decoder.More() {
		var entry disputetypes.VoterStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.DisputeId, entry.VoterAddress)
		if err := votersMap.Set(ctx, pair, *entry.Voter); err != nil {
			return err
		}
	}

	// Read closing bracket of voters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processReportersWithDelegatorsSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService) error {
	// Read "reporters_with_delegators_who_voted" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "reporters_with_delegators_who_voted" {
		return fmt.Errorf("expected reporters section, got %v", t)
	}

	// Read opening bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	sb := collections.NewSchemaBuilder(storeService)
	reportersWithDelsMap := collections.NewMap(sb,
		disputetypes.ReportersWithDelegatorsVotedBeforePrefix,
		"reporters_with_delegators_voted_before",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		sdk.IntValue,
	)
	// Read array values
	for decoder.More() {
		var entry disputetypes.ReportersWithDelegatorsWhoVotedStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.ReporterAddress, entry.DisputeId)
		if err := reportersWithDelsMap.Set(ctx, pair, entry.VotedAmount); err != nil {
			return err
		}
	}

	// Read closing bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processBlockInfoSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "block_info" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "block_info" {
		return fmt.Errorf("expected block_info section, got %v", t)
	}

	// Read opening bracket of block_info array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	sb := collections.NewSchemaBuilder(storeService)
	blockInfoMap := collections.NewMap(sb,
		disputetypes.BlockInfoPrefix,
		"block_info",
		collections.BytesKey,
		codec.CollValue[disputetypes.BlockInfo](cdc),
	)

	// Read array values
	for decoder.More() {
		var entry disputetypes.BlockInfoStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		if err := blockInfoMap.Set(ctx, entry.HashId, *entry.BlockInfo); err != nil {
			return err
		}
	}

	// Read closing bracket of block_info array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDisputeFeePayerSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "dispute_fee_payer" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "dispute_fee_payer" {
		return fmt.Errorf("expected dispute_fee_payer section, got %v", t)
	}

	// Read opening bracket of dispute_fee_payer array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	sb := collections.NewSchemaBuilder(storeService)
	disputeFeePayerMap := collections.NewMap(sb,
		disputetypes.DisputeFeePayerPrefix,
		"dispute_fee_payer",
		collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey),
		codec.CollValue[disputetypes.PayerInfo](cdc),
	)
	// Read array values
	for decoder.More() {
		var entry disputetypes.DisputeFeePayerStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.DisputeId, entry.Payer)
		if err := disputeFeePayerMap.Set(ctx, pair, *entry.PayerInfo); err != nil {
			return err
		}
	}

	// Read closing bracket of dispute_fee_payer array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDustSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService) error {
	// Read "dust" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "dust" {
		return fmt.Errorf("expected dust section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	dustMap := collections.NewItem(sb,
		disputetypes.DustKeyPrefix,
		"dust",
		sdk.IntValue,
	)
	// Read dust value
	t, err = decoder.Token()
	if err != nil {
		return err
	}

	// Convert string to math.Int
	dustStr, ok := t.(string)
	if !ok {
		return fmt.Errorf("expected dust value as string, got %v", t)
	}
	dust, ok := math.NewIntFromString(dustStr)
	if !ok {
		return fmt.Errorf("invalid dust value: %s", dustStr)
	}
	fmt.Println("Dust: ", dust)

	if err := dustMap.Set(ctx, dust); err != nil {
		return err
	}

	return nil
}

func processVoteCountsSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "vote_counts_by_group" property name
	t, err := decoder.Token()
	if err != nil {
		if err == io.EOF {
			return nil // Vote counts section is optional
		}
		return err
	}
	if name, ok := t.(string); !ok || name != "vote_counts_by_group" {
		return nil // Not the vote counts section, which is fine
	}

	// Read opening bracket of vote_counts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	sb := collections.NewSchemaBuilder(storeService)
	voteCountsMap := collections.NewMap(sb,
		disputetypes.VoteCountsByGroupPrefix,
		"vote_counts_by_group",
		collections.Uint64Key,
		codec.CollValue[disputetypes.StakeholderVoteCounts](cdc),
	)

	// Read array values
	for decoder.More() {
		var entry disputetypes.VoteCountsByGroupStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		if err := voteCountsMap.Set(ctx, entry.DisputeId, disputetypes.StakeholderVoteCounts{
			Users:     *entry.Users,
			Reporters: *entry.Reporters,
			Team:      *entry.Team,
		}); err != nil {
			return err
		}
	}

	// Read closing bracket of vote_counts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
