package fork

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

type AggregateStateData struct {
	Aggregate oracletypes.Aggregate `json:"aggregate"`
	Timestamp uint64                `json:"timestamp"`
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

	// Process TRB bridge aggregates array
	if err := processTrbBridgeAggregatesSection(ctx, decoder, sb, cdc); err != nil {
		return err
	}

	if err := processChecksumSection(decoder); err != nil {
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

	if err := expectColon(decoder); err != nil {
		return err
	}

	tipperTotalMap := collections.NewMap(sb,
		oracletypes.TipperTotalPrefix,
		"tipper_total",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		sdk.IntValue,
	)
	tOpen, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := tOpen.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected [ for tipper_total array, got %v", tOpen)
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
	name, err := readSectionKey(decoder)
	if err != nil {
		return err
	}
	if name != "latest_total_tips" {
		return fmt.Errorf("expected latest_total_tips section, got %s", name)
	}

	if err := expectColon(decoder); err != nil {
		return err
	}

	totalTipsMap := collections.NewMap(sb,
		oracletypes.TotalTipsPrefix,
		"total_tips",
		collections.Uint64Key,
		sdk.IntValue,
	)

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
	name, err := readSectionKey(decoder)
	if err != nil {
		return err
	}
	if name != "tipped_queries" {
		return fmt.Errorf("expected tipped_queries section, got %s", name)
	}

	if err := expectColon(decoder); err != nil {
		return err
	}

	tippedQueriesMap := collections.NewIndexedMap(sb,
		oracletypes.QueryTipPrefix,
		"query",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[oracletypes.QueryMeta](cdc),
		NewQueryIndex(sb),
	)

	tOpen, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := tOpen.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected [ for tipped_queries array, got %v", tOpen)
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

func processTrbBridgeAggregatesSection(ctx context.Context, decoder *json.Decoder, sb *collections.SchemaBuilder, cdc codec.BinaryCodec) error {
	name, err := readSectionKey(decoder)
	if err != nil {
		return err
	}
	if name != "trbbridge_aggregates" {
		return fmt.Errorf("expected trbbridge_aggregates section, got %s", name)
	}

	if err := expectColon(decoder); err != nil {
		return err
	}

	aggregateMap := collections.NewIndexedMap(sb,
		oracletypes.AggregatesPrefix,
		"aggregates",
		collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
		codec.CollValue[oracletypes.Aggregate](cdc),
		oracletypes.NewAggregatesIndex(sb),
	)
	noncesMap := collections.NewMap(sb,
		oracletypes.NoncesPrefix,
		"nonces",
		collections.BytesKey,
		collections.Uint64Value,
	)

	tOpen, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := tOpen.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected [ for trbbridge_aggregates array, got %v", tOpen)
	}

	maxNonceByQueryID := make(map[string]uint64)
	for decoder.More() {
		var entry AggregateStateData
		if err := decoder.Decode(&entry); err != nil {
			return err
		}

		pair := collections.Join(entry.Aggregate.QueryId, entry.Timestamp)
		if err := aggregateMap.Set(ctx, pair, entry.Aggregate); err != nil {
			return fmt.Errorf("failed to set aggregate: %w", err)
		}

		qKey := string(slices.Clone(entry.Aggregate.QueryId))
		if entry.Aggregate.Index > maxNonceByQueryID[qKey] {
			maxNonceByQueryID[qKey] = entry.Aggregate.Index
		}
	}

	if _, err := decoder.Token(); err != nil {
		return err
	}

	for qKey, n := range maxNonceByQueryID {
		if err := noncesMap.Set(ctx, []byte(qKey), n); err != nil {
			return fmt.Errorf("failed to set nonce for query id: %w", err)
		}
	}

	return nil
}

func processChecksumSection(decoder *json.Decoder) error {
	name, err := readSectionKey(decoder)
	if err != nil {
		return err
	}
	if name != "checksum" {
		return fmt.Errorf("expected checksum field, got %s", name)
	}
	if err := expectColon(decoder); err != nil {
		return err
	}
	var checksum string
	if err := decoder.Decode(&checksum); err != nil {
		return err
	}
	_ = checksum

	closeTok, err := decoder.Token()
	if err != nil {
		return err
	}
	closeBrace, ok := closeTok.(json.Delim)
	if !ok || closeBrace != '}' {
		return fmt.Errorf("expected closing brace after checksum, got %v", closeTok)
	}
	return nil
}

func expectColon(decoder *json.Decoder) error {
	t, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok || delim != ':' {
		return fmt.Errorf("expected ':', got %v", t)
	}
	return nil
}

// readSectionKey returns the next object property name, skipping comma delimiters between members.
func readSectionKey(decoder *json.Decoder) (string, error) {
	for {
		t, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch v := t.(type) {
		case json.Delim:
			if v == ',' {
				continue
			}
			return "", fmt.Errorf("unexpected delimiter while reading object key: %v", v)
		case string:
			return v, nil
		default:
			return "", fmt.Errorf("unexpected JSON token type %T", t)
		}
	}
}
