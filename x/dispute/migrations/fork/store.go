package fork

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
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

func MigrateStore(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	// Open the JSON file
	file, err := os.Open("dispute_module_state.json")
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
	if err := processDisputesSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process votes array
	if err := processVotesSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process voters array
	if err := processVotersSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process reporters with delegators array
	if err := processReportersWithDelegatorsSection(decoder, store); err != nil {
		return err
	}

	// Process block info array
	if err := processBlockInfoSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process dispute fee payer array
	if err := processDisputeFeePayerSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process dust value
	if err := processDustSection(decoder, store); err != nil {
		return err
	}

	// Process vote counts by group array (if present)
	if err := processVoteCountsSection(decoder, store, cdc); err != nil {
		return err
	}

	return nil
}

func processDisputesSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "disputes" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "disputes" {
		return fmt.Errorf("expected disputes section, got %v", t)
	}

	disputesStore := prefix.NewStore(store, disputetypes.DisputesPrefix)

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

		key := make([]byte, collections.Uint64Key.Size(8))
		collections.Uint64Key.Encode(key, entry.DisputeId)
		disputesStore.Set(key, cdc.MustMarshal(entry.Dispute))
	}

	// Read closing bracket of disputes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processVotesSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "votes" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "votes" {
		return fmt.Errorf("expected votes section, got %v", t)
	}

	votesStore := prefix.NewStore(store, disputetypes.VotesPrefix)

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
		key := make([]byte, collections.Uint64Key.Size(8))
		collections.Uint64Key.Encode(key, entry.DisputeId)
		votesStore.Set(key, cdc.MustMarshal(entry.Vote))
	}

	// Read closing bracket of votes array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processVotersSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "voters" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "voters" {
		return fmt.Errorf("expected voters section, got %v", t)
	}

	votersStore := prefix.NewStore(store, disputetypes.VoterVotePrefix)

	// Read opening bracket of voters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey)
	// Read array values
	for decoder.More() {
		var entry disputetypes.VoterStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.DisputeId, entry.VoterAddress)
		key := make([]byte, keyCodec.Size(pair))
		_, err = keyCodec.Encode(key, pair)
		if err != nil {
			panic(err)
		}
		votersStore.Set(key, cdc.MustMarshal(entry.Voter))
	}

	// Read closing bracket of voters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processReportersWithDelegatorsSection(decoder *json.Decoder, store storetypes.KVStore) error {
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

	reportersWithDelsStore := prefix.NewStore(store, disputetypes.ReportersWithDelegatorsVotedBeforePrefix)
	pairKeyCodec := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)

	// Read array values
	for decoder.More() {
		var entry disputetypes.ReportersWithDelegatorsWhoVotedStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.ReporterAddress, entry.DisputeId)
		key := make([]byte, pairKeyCodec.Size(pair))
		_, err = pairKeyCodec.Encode(key, pair)
		if err != nil {
			panic(err)
		}
		data, err := json.Marshal(entry.VotedAmount)
		if err != nil {
			return err
		}
		reportersWithDelsStore.Set(key, data)
	}

	// Read closing bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processBlockInfoSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
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

	blockInfoStore := prefix.NewStore(store, disputetypes.BlockInfoPrefix)
	keyCodec := collections.BytesKey

	// Read array values
	for decoder.More() {
		var entry disputetypes.BlockInfoStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		key := make([]byte, keyCodec.Size(entry.HashId))
		_, err = keyCodec.Encode(key, entry.HashId)
		if err != nil {
			panic(err)
		}
		blockInfoStore.Set(key, cdc.MustMarshal(entry.BlockInfo))
	}

	// Read closing bracket of block_info array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDisputeFeePayerSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
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

	disputeFeePayerStore := prefix.NewStore(store, disputetypes.DisputeFeePayerPrefix)
	keyCodec := collections.PairKeyCodec(collections.Uint64Key, collections.BytesKey)

	// Read array values
	for decoder.More() {
		var entry disputetypes.DisputeFeePayerStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		pair := collections.Join(entry.DisputeId, entry.Payer)
		key := make([]byte, keyCodec.Size(pair))
		_, err = keyCodec.Encode(key, pair)
		if err != nil {
			panic(err)
		}
		disputeFeePayerStore.Set(key, cdc.MustMarshal(entry.PayerInfo))
	}

	// Read closing bracket of dispute_fee_payer array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDustSection(decoder *json.Decoder, store storetypes.KVStore) error {
	// Read "dust" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "dust" {
		return fmt.Errorf("expected dust section, got %v", t)
	}

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

	dustStore := prefix.NewStore(store, disputetypes.DustKeyPrefix)
	data, err := json.Marshal(dust)
	if err != nil {
		panic(err)
	}
	dustStore.Set(disputetypes.DustKeyPrefix.Bytes(), data)

	return nil
}

func processVoteCountsSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
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

	voteCountsStore := prefix.NewStore(store, disputetypes.VoteCountsByGroupPrefix)
	keyCodec := collections.Uint64Key

	// Read array values
	for decoder.More() {
		var entry disputetypes.VoteCountsByGroupStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}
		key := make([]byte, keyCodec.Size(8))
		_, err = keyCodec.Encode(key, entry.DisputeId)
		if err != nil {
			panic(err)
		}
		voteCountsStore.Set(key, cdc.MustMarshal(&disputetypes.StakeholderVoteCounts{Users: *entry.Users, Reporters: *entry.Reporters, Team: *entry.Team}))
	}

	// Read closing bracket of vote_counts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
