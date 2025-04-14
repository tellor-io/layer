package keeper

import (
	"context"
	"errors"
	"fmt"
	gomath "math"

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

const (
	twelveHrsInMillis = 12 * 60 * 60 * 1000
)

type (
	Keeper struct {
		cdc                codec.BinaryCodec
		storeService       store.KVStoreService
		Params             collections.Item[types.Params]
		accountKeeper      types.AccountKeeper
		bankKeeper         types.BankKeeper
		bridgeKeeper       types.BridgeKeeper
		registryKeeper     types.RegistryKeeper
		reporterKeeper     types.ReporterKeeper
		Schema             collections.Schema
		CyclelistSequencer collections.Sequence                                                                                       // key: queryId, tipper
		TipperTotal        collections.Map[collections.Pair[[]byte, uint64], math.Int]                                                // key: tipperAcc, blockNumber
		TotalTips          collections.Map[uint64, math.Int]                                                                          // key: blockNumber
		Nonces             collections.Map[[]byte, uint64]                                                                            // key: queryId
		Reports            *collections.IndexedMap[collections.Triple[[]byte, []byte, uint64], types.MicroReport, types.ReportsIndex] // key: queryId, reporter, queryMeta.id
		QuerySequencer     collections.Sequence
		Query              *collections.IndexedMap[collections.Pair[[]byte, uint64], types.QueryMeta, types.QueryMetaIndex]  // key: queryId, id
		Aggregates         *collections.IndexedMap[collections.Pair[[]byte, uint64], types.Aggregate, types.AggregatesIndex] // key: queryId, timestamp
		Cyclelist          collections.Map[[]byte, []byte]                                                                   // key: queryId
		QueryDataLimit     collections.Item[types.QueryDataLimit]                                                            // query data bytes limit
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority      string
		Values         *collections.IndexedMap[collections.Pair[uint64, string], types.Value, types.ValuesIndex] // key: queryMeta.Id, valueHexstring  value: reporter's power
		AggregateValue collections.Map[uint64, types.RunningAggregate]                                           // key: queryMeta.Id
		// maintain a total weight for each querymeta.id
		ValuesWeightSum collections.Map[uint64, uint64] // key: queryMeta.Id value: totalWeight
		// storage for values that are aggregated via weighted mode
		ValuesWeightedMode collections.Map[collections.Pair[uint64, string], uint64] // key: queryMeta.Id, valueHexstring  value: total power of reporters that submitted the value
		// storage for bridge deposit reports queue
		BridgeDepositQueue collections.Map[collections.Pair[uint64, uint64], uint64] // key: aggregate timestamp, queryMetaId, value: depositId
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
		Values: collections.NewIndexedMap(sb,
			types.ValuesPrefix, "values",
			collections.PairKeyCodec(collections.Uint64Key, collections.StringKey),
			codec.CollValue[types.Value](cdc), types.NewValuesIndex(sb),
		),
		AggregateValue:  collections.NewMap(sb, types.AggregateValuePrefix, "aggregate_value", collections.Uint64Key, codec.CollValue[types.RunningAggregate](cdc)),
		ValuesWeightSum: collections.NewMap(sb, types.ValuesWeightSumPrefix, "values_weight_sum", collections.Uint64Key, collections.Uint64Value),
		// TotalTips maps the block number to the total tips added up till that point. Used for calculating voting power during a dispute
		TotalTips: collections.NewMap(sb, types.TotalTipsPrefix, "total_tips", collections.Uint64Key, sdk.IntValue),
		// Nonces maps the queryId to the nonce that increments with each aggregate report
		Nonces: collections.NewMap(sb, types.NoncesPrefix, "nonces", collections.BytesKey, collections.Uint64Value),
		// Aggregates maps the queryId:timestamp to the aggregate report plus indexes the key by the aggregate report's height
		// and the microReport's height (the microReport that becomes the median)
		// the microReport's height is needed to be able to flag the aggregate report in the case of a dispute
		Aggregates: collections.NewIndexedMap(sb,
			types.AggregatesPrefix,
			"aggregates",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.Aggregate](cdc),
			types.NewAggregatesIndex(sb),
		),
		// Reports maps the queryId:reporter:queryMeta.id to the microReport
		// indexes the key by the reporter (for a getter that gets all microReports by a reporter) and the queryMeta.id to fetch all microReports for a specific query during aggregation
		Reports: collections.NewIndexedMap(sb,
			types.ReportsPrefix,
			"reports",
			collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.MicroReport](cdc),
			types.NewReportsIndex(sb),
		),
		// QuerySequencer is an id generator for queryMeta that increments with each new query to distinguish between expired queries and new queries
		QuerySequencer: collections.NewSequence(sb, types.QuerySeqPrefix, "sequencer"),
		// Query maps the queryId:id to the queryMeta (holds information about the query and the tip, expiration time, tip amount, query spec reporting window etc.)
		// indexes the key by the query's queryType (ie SpotPrice, etc.) for purposes of updating the query's reporting spec (ie reporting block window)
		// also indexes by a boolean to distinguish between queries that have reports to be aggregated and not
		Query: collections.NewIndexedMap(sb,
			types.QueryTipPrefix,
			"query",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.QueryMeta](cdc),
			types.NewQueryIndex(sb),
		),
		// Cyclelist maps the queryId (hash of the query data) to the queryData for queries that are in the cycle list
		Cyclelist: collections.NewMap(sb, types.CyclelistPrefix, "cyclelist", collections.BytesKey, collections.BytesValue),
		// CyclelistSequencer is an id generator for cycle list queries that increments when called until the max of len(cycleListQueries) is reached
		// then it resets.
		CyclelistSequencer: collections.NewSequence(sb, types.CycleSeqPrefix, "cycle_sequencer"),
		// TipperTotal maps the tipperAcc:blockNumber to the total tips the tipper has added up till that point. Used for calculating voting power during a dispute
		TipperTotal: collections.NewMap(sb,
			types.TipperTotalPrefix,
			"tipper_total",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			sdk.IntValue,
		),
		// QueryDataLimit is the maximum number of bytes query data can be
		QueryDataLimit: collections.NewItem(sb, types.QueryDataLimitPrefix, "query_data_limit", codec.CollValue[types.QueryDataLimit](cdc)),
		// ClaimDepositQueue maps [metad, timestamp]:depositId
		BridgeDepositQueue: collections.NewMap(sb,
			types.BridgeDepositQueuePrefix,
			"bridge_deposit_queue",
			collections.PairKeyCodec(collections.Uint64Key, collections.Uint64Key),
			collections.Uint64Value),
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

func (k *Keeper) SetBridgeKeeper(bk types.BridgeKeeper) {
	k.bridgeKeeper = bk
}

// initialize query for a given query data.
// set the id, queryType, and reporting window
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
		Id:                      id,
		RegistrySpecBlockWindow: dataSpec.ReportBlockWindow,
		QueryData:               querydata,
	}
	return query, nil
}

func (k Keeper) CurrentQuery(ctx context.Context, queryId []byte) (query types.QueryMeta, err error) {
	err = k.Query.Walk(ctx, collections.NewPrefixedPairRange[[]byte, uint64](queryId).EndInclusive(gomath.MaxUint64).Descending(), func(key collections.Pair[[]byte, uint64], value types.QueryMeta) (bool, error) {
		query = value
		return true, nil
	})
	if err != nil {
		return types.QueryMeta{}, err
	}
	if query.QueryData == nil {
		return types.QueryMeta{}, collections.ErrNotFound
	}
	return query, nil
}

func (k Keeper) UpdateQuery(ctx context.Context, queryType string, newBlockWindow uint64) error {
	iter, err := k.Query.Indexes.QueryType.MatchExact(ctx, queryType)
	if err != nil {
		return err
	}

	queries, err := indexes.CollectValues(ctx, k.Query, iter)
	if err != nil {
		return err
	}
	for _, query := range queries {
		query.RegistrySpecBlockWindow = newBlockWindow
		queryId := utils.QueryIDFromData(query.QueryData)
		err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) FlagAggregateReport(ctx context.Context, report types.MicroReport) error {
	reporter := sdk.MustAccAddressFromBech32(report.Reporter)
	iter, err := k.Aggregates.Indexes.Reporter.MatchExact(ctx, collections.Join3(reporter.Bytes(), report.QueryId, report.BlockNumber))
	if err != nil {
		return err
	}
	defer iter.Close()
	if iter.Valid() {
		aggregatekey, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		aggregate, err := k.Aggregates.Get(ctx, aggregatekey)
		if err != nil {
			return err
		}
		aggregate.Flagged = true
		return k.Aggregates.Set(ctx, aggregatekey, aggregate)
	}
	return nil
}

func (k Keeper) ValidateMicroReportExists(ctx context.Context, reporter sdk.AccAddress, meta_id uint64, query_id []byte) (*types.MicroReport, bool, error) {
	report, err := k.Reports.Get(ctx, collections.Join3(query_id, reporter.Bytes(), meta_id))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return &report, true, nil
}

// iterate through DepositQueue
// Check if any deposit aggregate timestamp is >12 hrs old
// Call claim deposit on those deposits and remove from queue
// claim deposit should only fail if aggregate power is not reached, meaning deposit will need tipped again
// once tipped and reported for again, deposit should reenter the queue
func (k Keeper) AutoClaimDeposits(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlocktime := sdkCtx.BlockTime()
	thresholdTimestamp := uint64(currentBlocktime.UnixMilli() - twelveHrsInMillis)
	fmt.Println("threshold: ", thresholdTimestamp)

	// k1: timestamp, k2: metaId
	rng := collections.NewPrefixUntilPairRange[uint64, uint64](thresholdTimestamp)
	fmt.Println("rng: ", rng)

	var oldestDepositId uint64
	var aggregateTimestamp uint64
	var metaId uint64

	err := k.BridgeDepositQueue.Walk(ctx, rng, func(key collections.Pair[uint64, uint64], depositId uint64) (stop bool, err error) {
		oldestDepositId = depositId
		fmt.Println("oldestDepositId: ", oldestDepositId)
		aggregateTimestamp = key.K1()
		fmt.Println("k1: ", aggregateTimestamp)
		metaId = key.K2()
		fmt.Println("k2: ", metaId)
		return true, nil // Stop after the first (most recent) match
	})
	if err != nil {
		return err
	}

	if oldestDepositId == 0 && metaId == 0 {
		return nil
	}

	err = k.bridgeKeeper.ClaimDeposit(ctx, oldestDepositId, aggregateTimestamp)
	if err != nil {
		k.Logger(ctx).Error("autoClaimDeposits", "error", err)
		// remove the deposit from the queue if claiming fails
		err = k.BridgeDepositQueue.Remove(ctx, collections.Join(aggregateTimestamp, metaId))
		if err != nil {
			return err
		}
	}
	// remove the deposit from the queue after successful claim
	err = k.BridgeDepositQueue.Remove(ctx, collections.Join(aggregateTimestamp, metaId))
	if err != nil {
		return err
	}

	return nil
}
