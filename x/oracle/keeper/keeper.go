package keeper

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	regTypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var offset = time.Second * 3

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeService   store.KVStoreService
		Params         collections.Item[types.Params]
		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		registryKeeper types.RegistryKeeper
		reporterKeeper types.ReporterKeeper
		Schema         collections.Schema
		Commits        collections.Map[collections.Pair[[]byte, uint64], types.Commit]                      // key: reporter, queryid
		Tips           *collections.IndexedMap[collections.Pair[[]byte, []byte], math.Int, tipsIndex]       // key: queryId, tipper
		TipperTotal    *collections.IndexedMap[collections.Pair[[]byte, int64], math.Int, tipperTotalIndex] // key: tipperAcc, blockNumber
		// total tips given over time
		TotalTips          collections.Map[int64, math.Int]                                                                     // key: blockNumber, value: total tips                                  // key: queryId, timestamp
		Nonces             collections.Map[[]byte, uint64]                                                                      // key: queryId
		Reports            *collections.IndexedMap[collections.Triple[[]byte, []byte, uint64], types.MicroReport, reportsIndex] // key: queryId, reporter, query.id
		QuerySequencer     collections.Sequence
		Query              *collections.IndexedMap[[]byte, types.QueryMeta, queryMetaIndex]                           // key: queryId
		Aggregates         *collections.IndexedMap[collections.Pair[[]byte, int64], types.Aggregate, aggregatesIndex] // key: queryId, timestamp                                                                    // key: queryId                                                                  // keep track of the current cycle
		Cyclelist          collections.Map[[]byte, []byte]
		CyclelistSequencer collections.Sequence
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	registryKeeper types.RegistryKeeper,
	reporterKeeper types.ReporterKeeper,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params:         collections.NewItem(sb, types.ParamsKeyPrefix(), "params", codec.CollValue[types.Params](cdc)),
		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		registryKeeper: registryKeeper,
		reporterKeeper: reporterKeeper,

		authority: authority,

		Commits: collections.NewMap(sb, types.CommitsPrefix, "commits", collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key), codec.CollValue[types.Commit](cdc)),
		Tips: collections.NewIndexedMap(sb,
			types.TipsPrefix,
			"tips",
			collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			sdk.IntValue,
			NewTipsIndex(sb),
		),
		TotalTips:  collections.NewMap(sb, types.TotalTipsPrefix, "total_tips", collections.Int64Key, sdk.IntValue),
		Nonces:     collections.NewMap(sb, types.NoncesPrefix, "nonces", collections.BytesKey, collections.Uint64Value),
		Aggregates: collections.NewIndexedMap(sb, types.AggregatesPrefix, "aggregates", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), codec.CollValue[types.Aggregate](cdc), NewAggregatesIndex(sb)),
		Reports: collections.NewIndexedMap(sb,
			types.ReportsPrefix,
			"reports",
			collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.MicroReport](cdc),
			NewReportsIndex(sb),
		),
		QuerySequencer: collections.NewSequence(sb, types.QuerySeqPrefix, "sequencer"),
		Query: collections.NewIndexedMap(sb,
			types.QueryTipPrefix,
			"query",
			collections.BytesKey,
			codec.CollValue[types.QueryMeta](cdc),
			NewQueryIndex(sb),
		),
		Cyclelist:          collections.NewMap(sb, types.CyclelistPrefix, "cyclelist", collections.BytesKey, collections.BytesValue),
		CyclelistSequencer: collections.NewSequence(sb, types.CycleSeqPrefix, "cycle_sequencer"),

		TipperTotal: collections.NewIndexedMap(sb,
			types.TipperTotalPrefix,
			"tipper_total",
			collections.PairKeyCodec(collections.BytesKey, collections.Int64Key),
			sdk.IntValue,
			NewTippersIndex(sb),
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// initialize query for a given query data
func (k Keeper) InitializeQuery(ctx context.Context, querydata []byte) (types.QueryMeta, error) {
	// initialize query tip first time

	queryType, _, err := regTypes.DecodeQueryType(querydata)
	if err != nil {
		return types.QueryMeta{}, err
	}
	dataSpec, err := k.GetDataSpec(ctx, queryType)
	if err != nil {
		return types.QueryMeta{}, err
	}
	id, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return types.QueryMeta{}, err
	}
	query := types.QueryMeta{
		Id:                    id,
		RegistrySpecTimeframe: dataSpec.ReportBufferWindow,
		QueryId:               utils.QueryIDFromData(querydata),
	}
	return query, nil
}

func (k Keeper) UpdateQuery(ctx context.Context, queryType string, newTimeframe time.Duration) error {
	iter, err := k.Query.Indexes.QueryType.MatchExact(ctx, queryType)
	if err != nil {
		return err
	}

	queries, err := indexes.CollectValues(ctx, k.Query, iter)
	if err != nil {
		return err
	}
	for _, query := range queries {
		query.RegistrySpecTimeframe = newTimeframe
		err = k.Query.Set(ctx, query.QueryId, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) FlagAggregateReport(ctx context.Context, report types.MicroReport) error {
	iter, err := k.Aggregates.Indexes.MicroHeight.MatchExact(ctx, report.BlockNumber)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		aggregatekey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		if bytes.Equal(aggregatekey.K1(), report.QueryId) {
			aggregate, err := k.Aggregates.Get(ctx, aggregatekey)
			if err != nil {
				return err
			}
			reporter := aggregate.Reporters[aggregate.AggregateReportIndex].Reporter
			if sdk.MustAccAddressFromBech32(reporter).Equals(sdk.MustAccAddressFromBech32(report.Reporter)) {
				aggregate.Flagged = true
				return k.Aggregates.Set(ctx, aggregatekey, aggregate)
			}
		}
	}

	return nil
}
