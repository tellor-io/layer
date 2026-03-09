package keeper

import (
	"context"
	"errors"
	"fmt"
	gomath "math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	regTypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	// twoMinInMillis    = 2 * 60 * 1000
)

type (
	Keeper struct {
		cdc                codec.BinaryCodec
		storeService       store.KVStoreService
		Params             collections.Item[types.Params]
		MaxBatchSize       collections.Item[uint32]
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
		NoStakeReports     *collections.IndexedMap[collections.Pair[[]byte, uint64], types.NoStakeMicroReport, types.ReporterIndex]   // key: queryId, timestamp
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
		BridgeDepositQueue collections.Map[collections.Pair[uint64, uint64], []byte] // key: aggregate timestamp, queryMetaId, value: queryData
		// storage for no stake report queryId / queryData
		NoStakeReportedQueries collections.Map[[]byte, []byte] // key: queryId, value: queryData

		// Liveness reward storage
		CycleCount            collections.Sequence                                              // tracks completed cycles
		Dust                  collections.Item[math.Int]                                        // leftover from rounding during distribution
		QueryOpportunities    collections.Map[[]byte, uint64]                                   // key: queryId, value: opportunity count
		ReporterQueryShareSum collections.Map[collections.Pair[[]byte, []byte], math.LegacyDec] // key: (reporter, queryId), value: share sum

		// Liveness tracking (standard/non-standard split)
		ReporterStandardShareSum collections.Map[[]byte, math.LegacyDec] // key: reporter, value: sum of shares for standard queries
		NonStandardQueries       collections.Map[[]byte, bool]           // key: queryId, value: true if non-standard
		StandardOpportunities    collections.Item[uint64]                // number of opportunities for standard queries (cycles completed)

		// Liveness tracking - for percent liveness query
		TotalAggregatesCount    collections.Item[uint64]        // total aggregates on chain
		ReporterAggregatesCount collections.Map[[]byte, uint64] // key: reporter, value: reports submitted (incremented at submission)
		ReporterLastReportTime  collections.Map[[]byte, int64]  // key: reporter, value: timestamp of last report
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
		MaxBatchSize:   collections.NewItem(sb, types.MaxBatchSizePrefix, "max_batch_size", collections.Uint32Value),
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
		// NoStakeReports maps the queryId:reporter:timestamp to the microReport
		NoStakeReports: collections.NewIndexedMap(sb,
			types.NoStakeReportsPrefix,
			"no_stake_reports",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.NoStakeMicroReport](cdc),
			types.NewReporterIndex(sb),
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
			collections.BytesValue),
		// NoStakeReportedQueries maps the queryId to the queryData
		NoStakeReportedQueries: collections.NewMap(sb, types.NoStakeReportedQueriesPrefix, "no_stake_reported_queries", collections.BytesKey, collections.BytesValue),

		// Liveness reward storage
		CycleCount:            collections.NewSequence(sb, types.CycleCountPrefix, "cycle_count"),
		Dust:                  collections.NewItem(sb, types.DustPrefix, "dust", sdk.IntValue),
		QueryOpportunities:    collections.NewMap(sb, types.QueryOpportunitiesPrefix, "query_opportunities", collections.BytesKey, collections.Uint64Value),
		ReporterQueryShareSum: collections.NewMap(sb, types.ReporterQueryShareSumPrefix, "reporter_query_share_sum", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), layertypes.LegacyDecValue),

		// Optimized liveness tracking initialization
		ReporterStandardShareSum: collections.NewMap(sb, types.ReporterStandardShareSumPrefix, "reporter_standard_share_sum", collections.BytesKey, layertypes.LegacyDecValue),
		NonStandardQueries:       collections.NewMap(sb, types.NonStandardQueriesPrefix, "non_standard_queries", collections.BytesKey, collections.BoolValue),
		StandardOpportunities:    collections.NewItem(sb, types.StandardOpportunitiesPrefix, "standard_opportunities", collections.Uint64Value),

		// Liveness tracking
		TotalAggregatesCount:    collections.NewItem(sb, types.TotalAggregatesCountPrefix, "total_aggregates_count", collections.Uint64Value),
		ReporterAggregatesCount: collections.NewMap(sb, types.ReporterAggregatesCountPrefix, "reporter_aggregates_count", collections.BytesKey, collections.Uint64Value),
		ReporterLastReportTime:  collections.NewMap(sb, types.ReporterLastReportTimePrefix, "reporter_last_report_time", collections.BytesKey, collections.Int64Value),
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
	if strings.EqualFold(queryType, TRBBridgeQueryType) {
		return types.QueryMeta{}, errors.New("cannot initialize deprecated TRBBridge query type")
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

// ranger checks for any timestamps in queue > 12 hrs old
// call claim deposit on the oldest deposit aggregate and remove from queue
// claim deposit should only fail if aggregate power is not reached, meaning deposit will need tipped again
// once tipped and reported for again, deposit will reenter the queue
func (k Keeper) AutoClaimDeposits(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentBlocktime := sdkCtx.BlockTime()
	thresholdTimestamp := uint64(currentBlocktime.UnixMilli() - twelveHrsInMillis)

	// ranger only finds timestamps exactly 12 hrs and older
	rng := collections.NewPrefixUntilPairRange[uint64, uint64](thresholdTimestamp)

	var queryData []byte
	var aggregateTimestamp uint64
	var metaId uint64

	err := k.BridgeDepositQueue.Walk(ctx, rng, func(key collections.Pair[uint64, uint64], bz []byte) (stop bool, err error) {
		aggregateTimestamp = key.K1()
		metaId = key.K2()
		queryData = bz
		return true, nil // stop after the first (oldest) match
	})
	if err != nil {
		k.Logger(ctx).Error("autoClaimDeposits", "error walking through queue", err)
		return err
	}
	// if no matches, return nil
	if queryData == nil && metaId == 0 {
		return nil
	}

	// decode retreieved query data
	depositId, err := k.DecodeBridgeDeposit(ctx, queryData)
	if err != nil {
		k.Logger(ctx).Error("autoClaimDeposits", "error decoding query data", err)
		return err
	}

	err = k.bridgeKeeper.ClaimDeposit(ctx, depositId, aggregateTimestamp)
	if err != nil {
		k.Logger(ctx).Error("autoClaimDeposits", "error calling claim deposit", err)
		// remove the deposit from the queue if claiming fails
		err = k.BridgeDepositQueue.Remove(ctx, collections.Join(aggregateTimestamp, metaId))
		if err != nil {
			k.Logger(ctx).Error("autoClaimDeposits", "error removing bridge deposit from queue after failed claim", err)
			return err
		}
	}
	// remove the deposit from the queue after successful claim
	err = k.BridgeDepositQueue.Remove(ctx, collections.Join(aggregateTimestamp, metaId))
	if err != nil {
		k.Logger(ctx).Error("autoClaimDeposits", "error removing bridge deposit from queue after successful claim", err)
		return err
	}

	return nil
}

func (k Keeper) DecodeBridgeDeposit(ctx context.Context, queryData []byte) (uint64, error) {
	_, bytesArgs, err := regTypes.DecodeQueryType(queryData)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query type: %v", err))
	}
	// decode query data arguments
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return 0, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return 0, err
	}
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsDecoded, err := queryDataArgs.Unpack(bytesArgs)
	if err != nil {
		return 0, err
	}
	depositId := queryDataArgsDecoded[1].(*big.Int).Uint64()

	return depositId, nil
}

func (k Keeper) GetLastReportedAtTimestamp(ctx context.Context, reporter []byte) (uint64, error) {
	// get the last block they reported at
	reportedAtBlock, err := k.reporterKeeper.GetLastReportedAtBlock(ctx, reporter)
	if err != nil {
		return 0, errors.New("error getting last reported block: " + err.Error())
	}

	// get the timestamp of the report at that block
	rng := collections.NewPrefixUntilPairRange[uint64, collections.Pair[[]byte, uint64]](reportedAtBlock).Descending()
	iter, err := k.Aggregates.Indexes.BlockHeight.Iterate(ctx, rng)
	if err != nil {
		return 0, errors.New("error iterating over aggregate reports: " + err.Error())
	}
	defer iter.Close()

	// pull timestamp from the aggregate report key at given height
	var timestamp uint64
	if iter.Valid() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return 0, errors.New("error getting primary key: " + err.Error())
		}
		timestamp = key.K2()
	}

	if timestamp == 0 && reportedAtBlock != 0 {
		return 0, errors.New("no reports found")
	}

	return timestamp, nil
}

func (k Keeper) GetMaxBatchSize(ctx context.Context) (uint32, error) {
	maxBatchSize, err := k.MaxBatchSize.Get(ctx)
	// gets error only if not found, so just return 0
	if err != nil {
		return 0, nil
	}
	return maxBatchSize, nil
}

// GetTimestampForBlockHeight returns the timestamp of aggregates at a given block height.
// Returns 0 if no aggregates exist at that block height.
func (k Keeper) GetTimestampForBlockHeight(ctx context.Context, blockHeight uint64) (uint64, error) {
	rng := collections.NewPrefixedPairRange[uint64, collections.Pair[[]byte, uint64]](blockHeight)
	iter, err := k.Aggregates.Indexes.BlockHeight.Iterate(ctx, rng)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	if iter.Valid() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return 0, err
		}
		return key.K2(), nil
	}

	return 0, nil
}
