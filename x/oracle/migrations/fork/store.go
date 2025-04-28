package fork

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type TipperTotalData struct {
	TipperTotal math.Int
	Address     []byte
	Block       uint64
}

type TotalTipsData struct {
	TotalTips math.Int
	Block     uint64
}

type ModuleStateData struct {
	TipperTotal     []TipperTotalData       `json:"tipper_total"`
	LatestTotalTips TotalTipsData           `json:"total_tips"`
	TippedQueries   []oracletypes.QueryMeta `json:"tipped_queries"`
}

func MigrateFork(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec) error {
	store := runtime.KVStoreAdapter(storeService.OpenKVStore(ctx))

	// Open the JSON file
	file, err := os.Open("oracle_module_state.json")
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

	// Process tipper total array
	if err := processTipperTotalSection(decoder, store); err != nil {
		return err
	}

	// Process total tips array
	if err := processTotalTipsSection(decoder, store); err != nil {
		return err
	}

	// Process tipped queries array
	if err := processTippedQueriesSection(decoder, store, cdc); err != nil {
		return err
	}

	return nil
}

func processTipperTotalSection(decoder *json.Decoder, store storetypes.KVStore) error {
	// Read "tipper_total" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "tipper_total" {
		return fmt.Errorf("expected tipper_total section, got %v", t)
	}

	tipperTotalStore := prefix.NewStore(store, oracletypes.TipperTotalPrefix)

	// Read opening bracket of tipper_total array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)

	// Read array values
	for decoder.More() {
		var entry TipperTotalData
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		pair := collections.Join(entry.Address, entry.Block)
		key := make([]byte, keyCodec.Size(pair))
		_, err = keyCodec.Encode(key, pair)
		if err != nil {
			panic(err)
		}
		data, err := json.Marshal(entry.TipperTotal)
		if err != nil {
			panic(err)
		}
		tipperTotalStore.Set(key, data)
	}

	// Read closing bracket of tipper_total array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processTotalTipsSection(decoder *json.Decoder, store storetypes.KVStore) error {
	// Read "total_tips" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "total_tips" {
		return fmt.Errorf("expected total_tips section, got %v", t)
	}

	totalTipsStore := prefix.NewStore(store, oracletypes.TotalTipsPrefix)

	// Read and decode the single TotalTipsData object
	var entry TotalTipsData
	if err := decoder.Decode(&entry); err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, entry.Block)
	data, err := json.Marshal(entry.TotalTips)
	if err != nil {
		panic(err)
	}
	totalTipsStore.Set(key, data)

	return nil
}

func processTippedQueriesSection(decoder *json.Decoder, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	// Read "tipped_queries" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "tipped_queries" {
		return fmt.Errorf("expected tipped_queries section, got %v", t)
	}

	tippedQueriesStore := prefix.NewStore(store, []byte("tipped_queries"))

	// Read opening bracket of tipped_queries array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	keyCodec := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)

	// Read array values
	for decoder.More() {
		var entry oracletypes.QueryMeta
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		queryId := utils.QueryIDFromData(entry.QueryData)

		pair := collections.Join(queryId, entry.Id)
		key := make([]byte, keyCodec.Size(pair))
		_, err = keyCodec.Encode(key, pair)
		if err != nil {
			panic(err)
		}
		data, err := json.Marshal(entry)
		if err != nil {
			panic(err)
		}
		tippedQueriesStore.Set(key, data)
	}

	// Read closing bracket of tipped_queries array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
