package fork

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/tellor-io/layer/x/reporter/types"

	layertypes "github.com/tellor-io/layer/types"
)

type ReporterStateEntry struct {
	ReporterAddr []byte               `json:"reporter_addr"`
	Reporter     types.OracleReporter `json:"reporter"`
}

type SelectorStateEntry struct {
	SelectorAddr []byte          `json:"selector_addr"`
	Selector     types.Selection `json:"selector"`
}

type SelectorTipsStateEntry struct {
	SelectorAddress []byte `json:"selector_address"`
	Tips            string `json:"tips"`
}

type DisputedDelegationAmountStateEntry struct {
	HashId           []byte                   `json:"hash_id"`
	DelegationAmount types.DelegationsAmounts `json:"delegation_amount"`
}

type FeePaidFromStakeStateEntry struct {
	HashId           []byte                   `json:"hash_id"`
	DelegationAmount types.DelegationsAmounts `json:"delegation_amount"`
}

func MigrateFork(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec, pathToFile string) error {

	path := filepath.Join(
		pathToFile,
		"reporter_module_state.json",
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

	// Process reporters array
	if err := processReportersSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process selectors array
	if err := processSelectorsSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process selector tips array
	if err := processSelectorTipsSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process disputed delegation amounts array
	if err := processDisputedDelegationAmountsSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	// Process fee paid from stake array
	if err := processFeePaidFromStakeSection(ctx, decoder, storeService, cdc); err != nil {
		return err
	}

	return nil
}

func processReportersSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "reporters" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "reporters" {
		return fmt.Errorf("expected reporters section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	reporterStore := collections.NewMap(sb, types.ReportersKey, "reporters", collections.BytesKey, codec.CollValue[types.OracleReporter](cdc))

	// Read opening bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry ReporterStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		if err := reporterStore.Set(ctx, entry.ReporterAddr, entry.Reporter); err != nil {
			return err
		}
	}

	// Read closing bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

type ReporterSelectorsIndex struct {
	Reporter *indexes.Multi[[]byte, []byte, types.Selection]
}

func (a ReporterSelectorsIndex) IndexesList() []collections.Index[[]byte, types.Selection] {
	return []collections.Index[[]byte, types.Selection]{a.Reporter}
}

// maps a reporter address to its selectors' addresses
func NewSelectorsIndex(sb *collections.SchemaBuilder) ReporterSelectorsIndex {
	return ReporterSelectorsIndex{
		Reporter: indexes.NewMulti(
			sb, types.ReporterSelectorsIndexPrefix, "reporter_selectors_index",
			collections.BytesKey, collections.BytesKey,
			func(k []byte, del types.Selection) ([]byte, error) {
				return del.Reporter, nil
			},
		),
	}
}

func processSelectorsSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "selectors" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "selectors" {
		return fmt.Errorf("expected selectors section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	selectorIdxMap := collections.NewIndexedMap(sb, types.SelectorsKey, "selectors", collections.BytesKey, codec.CollValue[types.Selection](cdc), NewSelectorsIndex(sb))

	// Read opening bracket of selectors array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry SelectorStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		if err := selectorIdxMap.Set(ctx, entry.SelectorAddr, entry.Selector); err != nil {
			return err
		}
	}

	// Read closing bracket of selectors array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processSelectorTipsSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "selector_tips" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "selector_tips" {
		return fmt.Errorf("expected selector_tips section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	selectorTipsStore := collections.NewMap(sb, types.SelectorTipsPrefix, "selector_tips", collections.BytesKey, layertypes.LegacyDecValue)

	// Read opening bracket of selector_tips array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry SelectorTipsStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		tips, err := math.LegacyNewDecFromStr(entry.Tips)
		if err != nil {
			return err
		}

		if err := selectorTipsStore.Set(ctx, entry.SelectorAddress, tips); err != nil {
			return err
		}
	}

	// Read closing bracket of selector_tips array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDisputedDelegationAmountsSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "disputed_delegation_amounts" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "disputed_delegation_amounts" {
		return fmt.Errorf("expected disputed_delegation_amounts section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	disputedDelegationStore := collections.NewMap(sb, types.DisputedDelegationAmountsPrefix, "disputed_delegation_amounts", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc))

	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry DisputedDelegationAmountStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		if err := disputedDelegationStore.Set(ctx, entry.HashId, entry.DelegationAmount); err != nil {
			return err
		}
	}

	// Read closing bracket of disputed_delegation_amounts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processFeePaidFromStakeSection(ctx context.Context, decoder *json.Decoder, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	// Read "fee_paid_from_stake" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "fee_paid_from_stake" {
		return fmt.Errorf("expected fee_paid_from_stake section, got %v", t)
	}

	sb := collections.NewSchemaBuilder(storeService)
	feePaidStore := collections.NewMap(sb, types.FeePaidFromStakePrefix, "fee_paid_from_stake", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc))

	// Read opening bracket of fee_paid_from_stake array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	// Read array values
	for decoder.More() {
		var entry FeePaidFromStakeStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		if err := feePaidStore.Set(ctx, entry.HashId, entry.DelegationAmount); err != nil {
			return err
		}
	}

	// Read closing bracket of fee_paid_from_stake array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
