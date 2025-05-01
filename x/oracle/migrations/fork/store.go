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
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

type TipperTotalData struct {
	TipperTotal string `json:"tipper_total"`
	Address     []byte `json:"address"`
	Block       uint64 `json:"block"`
}

type TotalTipsData struct {
	TotalTips string `json:"total_tips"`
	Block     uint64 `json:"block"`
}

type ModuleStateData struct {
	TipperTotal     []TipperTotalData       `json:"tipper_total"`
	LatestTotalTips TotalTipsData           `json:"total_tips"`
	TippedQueries   []oracletypes.QueryMeta `json:"tipped_queries"`
}

func MigrateFork(ctx context.Context, storeService store.KVStoreService, cdc codec.BinaryCodec, pathToFile string) error {
	sb := collections.NewSchemaBuilder(storeService)
	path := filepath.Join(
		pathToFile,
		"oracle_module_state.json",
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

	// Process tipper total array
	if err := processTipperTotalSection(ctx, decoder, sb, cdc); err != nil {
		return err
	}

	// Process total tips array
	if err := processTotalTipsSection(ctx, decoder, sb, cdc); err != nil {
		return err
	}

	// Process tipped queries array
	if err := processTippedQueriesSection(ctx, decoder, sb, cdc); err != nil {
		return err
	}

	return nil
}

func processTipperTotalSection(ctx context.Context, decoder *json.Decoder, sb *collections.SchemaBuilder, cdc codec.BinaryCodec) error {
	// Read "tipper_total" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "tipper_total" {
		return fmt.Errorf("expected tipper_total section, got %v", t)
	}

	tipperTotalMap := collections.NewMap(sb,
		oracletypes.TipperTotalPrefix,
		"tipper_total",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		sdk.IntValue,
	)
	// Read opening bracket of tipper_total array
	if _, err := decoder.Token(); err != nil {
		return err
	}
	// Read array values
	for decoder.More() {
		var entry TipperTotalData
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		pair := collections.Join(entry.Address, entry.Block)
		tipTotal, ok := math.NewIntFromString(entry.TipperTotal)
		if !ok {
			return fmt.Errorf("cannot convert tipper total to int")
		}
		if err := tipperTotalMap.Set(ctx, pair, tipTotal); err != nil {
			return fmt.Errorf("failed to set tipper total: %w", err)
		}
	}

	// Read closing bracket of tipper_total array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}

func processTotalTipsSection(ctx context.Context, decoder *json.Decoder, sb *collections.SchemaBuilder, cdc codec.BinaryCodec) error {
	// Read "total_tips" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "latest_total_tips" {
		return fmt.Errorf("expected total_tips section, got %v", t)
	}

	totalTipsMap := collections.NewMap(sb,
		oracletypes.TotalTipsPrefix,
		"total_tips",
		collections.Uint64Key,
		sdk.IntValue,
	)

	// Read and decode the single TotalTipsData object
	var entry TotalTipsData
	if err := decoder.Decode(&entry); err != nil {
		return err
	}

	totalTips, ok := math.NewIntFromString(entry.TotalTips)
	if !ok {
		return fmt.Errorf("cannot convert tipper total to int")
	}
	if err := totalTipsMap.Set(ctx, entry.Block, totalTips); err != nil {
		return fmt.Errorf("failed to set total tips: %w", err)
	}

	return nil
}

type QueryMetaIndex struct {
	Expiration *indexes.Multi[collections.Pair[bool, uint64], collections.Pair[[]byte, uint64], oracletypes.QueryMeta]
	QueryType  *indexes.Multi[string, collections.Pair[[]byte, uint64], oracletypes.QueryMeta]
}

func (a QueryMetaIndex) IndexesList() []collections.Index[collections.Pair[[]byte, uint64], oracletypes.QueryMeta] {
	return []collections.Index[collections.Pair[[]byte, uint64], oracletypes.QueryMeta]{a.Expiration, a.QueryType}
}

func NewQueryIndex(sb *collections.SchemaBuilder) QueryMetaIndex {
	return QueryMetaIndex{
		Expiration: indexes.NewMulti(
			sb, oracletypes.QueryByExpirationPrefix, "query_by_expiration",
			collections.PairKeyCodec(collections.BoolKey, collections.Uint64Key), collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v oracletypes.QueryMeta) (collections.Pair[bool, uint64], error) {
				return collections.Join(v.HasRevealedReports, v.Expiration), nil
			},
		),
		QueryType: indexes.NewMulti(
			sb, oracletypes.QueryTypeIndexPrefix, "query_by_type",
			collections.StringKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			func(_ collections.Pair[[]byte, uint64], v oracletypes.QueryMeta) (string, error) {
				return v.QueryType, nil
			},
		),
	}
}

func processTippedQueriesSection(ctx context.Context, decoder *json.Decoder, sb *collections.SchemaBuilder, cdc codec.BinaryCodec) error {
	// Read "tipped_queries" property name
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	if name, ok := t.(string); !ok || name != "tipped_queries" {
		return fmt.Errorf("expected tipped_queries section, got %v", t)
	}

	tippedQueriesMap := collections.NewIndexedMap(sb,
		oracletypes.QueryTipPrefix,
		"query",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[oracletypes.QueryMeta](cdc),
		NewQueryIndex(sb),
	)

	// Read opening bracket of tipped_queries array
	if _, err := decoder.Token(); err != nil {
		return err
	}
	// Read array values
	for decoder.More() {
		var entry oracletypes.QueryMeta
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		queryId := utils.QueryIDFromData(entry.QueryData)

		pair := collections.Join(queryId, entry.Id)
		if err := tippedQueriesMap.Set(ctx, pair, entry); err != nil {
			return fmt.Errorf("failed to set tipped query: %w", err)
		}
	}

	// Read closing bracket of tipped_queries array
	if _, err := decoder.Token(); err != nil {
		return err
	}

	return nil
}
