package fork

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
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
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

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
	if err := processReportersSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process selectors array
	if err := processSelectorsSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process selector tips array
	if err := processSelectorTipsSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process disputed delegation amounts array
	if err := processDisputedDelegationAmountsSection(decoder, store, cdc); err != nil {
		return err
	}

	// Process fee paid from stake array
	if err := processFeePaidFromStakeSection(decoder, store, cdc); err != nil {
		return err
	}

	return nil
}

func processReportersSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "reporters" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "reporters" {
		return fmt.Errorf("expected reporters section, got %v", t)
	}

	reporterStore := prefix.NewStore(store, types.ReportersKey)
	keyCodec := collections.BytesKey
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

		key := make([]byte, keyCodec.Size(entry.ReporterAddr))
		keyCodec.Encode(key, entry.ReporterAddr)
		fmt.Println("in store.go key: ", hex.EncodeToString(key))
		data, err := cdc.Marshal(&entry.Reporter)
		if err != nil {
			return err
		}
		reporterStore.Set(key, data)
	}

	// Read closing bracket of reporters array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processSelectorsSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "selectors" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "selectors" {
		return fmt.Errorf("expected selectors section, got %v", t)
	}

	selectorStore := prefix.NewStore(store, types.SelectorsKey)

	// Read opening bracket of selectors array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.BytesKey

	// Read array values
	for decoder.More() {
		var entry SelectorStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		key := make([]byte, keyCodec.Size(entry.SelectorAddr))
		keyCodec.Encode(key, entry.SelectorAddr)

		data, err := cdc.Marshal(&entry.Selector)
		if err != nil {
			return err
		}
		selectorStore.Set(key, data)
	}

	// Read closing bracket of selectors array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processSelectorTipsSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "selector_tips" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "selector_tips" {
		return fmt.Errorf("expected selector_tips section, got %v", t)
	}

	selectorTipsStore := prefix.NewStore(store, types.SelectorTipsPrefix)

	// Read opening bracket of selector_tips array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.BytesKey

	// Read array values
	for decoder.More() {
		var entry SelectorTipsStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		key := make([]byte, keyCodec.Size(entry.SelectorAddress))
		keyCodec.Encode(key, entry.SelectorAddress)

		tips := math.LegacyMustNewDecFromStr(entry.Tips)
		data, err := layertypes.LegacyDecValue.Encode(tips)
		if err != nil {
			return err
		}
		selectorTipsStore.Set(key, data)
	}

	// Read closing bracket of selector_tips array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processDisputedDelegationAmountsSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "disputed_delegation_amounts" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "disputed_delegation_amounts" {
		return fmt.Errorf("expected disputed_delegation_amounts section, got %v", t)
	}

	disputedDelegationStore := prefix.NewStore(store, types.DisputedDelegationAmountsPrefix)

	// Read opening bracket of disputed_delegation_amounts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.BytesKey

	// Read array values
	for decoder.More() {
		var entry DisputedDelegationAmountStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		key := make([]byte, keyCodec.Size(entry.HashId))
		keyCodec.Encode(key, entry.HashId)

		data, err := cdc.Marshal(&entry.DelegationAmount)
		if err != nil {
			return err
		}
		disputedDelegationStore.Set(key, data)
	}

	// Read closing bracket of disputed_delegation_amounts array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processFeePaidFromStakeSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "fee_paid_from_stake" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "fee_paid_from_stake" {
		return fmt.Errorf("expected fee_paid_from_stake section, got %v", t)
	}

	feePaidStore := prefix.NewStore(store, types.FeePaidFromStakePrefix)

	// Read opening bracket of fee_paid_from_stake array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.BytesKey

	// Read array values
	for decoder.More() {
		var entry FeePaidFromStakeStateEntry
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		key := make([]byte, keyCodec.Size(entry.HashId))
		keyCodec.Encode(key, entry.HashId)

		data, err := cdc.Marshal(&entry.DelegationAmount)
		if err != nil {
			return err
		}
		feePaidStore.Set(key, data)
	}

	// Read closing bracket of fee_paid_from_stake array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
