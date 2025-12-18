package keeper

import (
	"bytes"
	"context"
	"fmt"
	"time"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	Keeper struct {
		cdc                       codec.BinaryCodec
		storeService              store.KVStoreService
		Params                    collections.Item[types.Params]
		Tracker                   collections.Item[types.StakeTracker]
		Reporters                 collections.Map[[]byte, types.OracleReporter]                                                                                             // key: reporter AccAddress
		SelectorTips              collections.Map[[]byte, math.LegacyDec]                                                                                                   // key: selector AccAddress
		Selectors                 *collections.IndexedMap[[]byte, types.Selection, ReporterSelectorsIndex]                                                                  // key: selector AccAddress
		DisputedDelegationAmounts collections.Map[[]byte, types.DelegationsAmounts]                                                                                         // key: dispute hashId
		FeePaidFromStake          collections.Map[[]byte, types.DelegationsAmounts]                                                                                         // key: dispute hashId
		Report                    *collections.IndexedMap[collections.Pair[[]byte, collections.Pair[[]byte, uint64]], types.DelegationsAmounts, ReporterBlockNumberIndexes] // key: queryId, (reporter AccAddress, blockNumber)

		// Distribution queue collections
		ReporterPeriodData       collections.Map[[]byte, types.PeriodRewardData]      // key: reporter AccAddress -> current period data
		DistributionQueue        collections.Map[uint64, types.DistributionQueueItem] // key: queue index -> item to distribute
		DistributionQueueCounter collections.Item[types.DistributionQueueCounter]     // tracks head and tail of queue

		Schema collections.Schema
		logger log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper  types.AccountKeeper
		stakingKeeper  types.StakingKeeper
		bankKeeper     types.BankKeeper
		registryKeeper types.RegistryKeeper
		oracleKeeper   types.OracleKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	accountKeeper types.AccountKeeper,
	stakingKeeper types.StakingKeeper,
	bankKeeper types.BankKeeper,
	registryKeeper types.RegistryKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params:                    collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Tracker:                   collections.NewItem(sb, types.StakeTrackerPrefix, "tracker", codec.CollValue[types.StakeTracker](cdc)),
		Reporters:                 collections.NewMap(sb, types.ReportersKey, "reporters", collections.BytesKey, codec.CollValue[types.OracleReporter](cdc)),
		Selectors:                 collections.NewIndexedMap(sb, types.SelectorsKey, "selectors", collections.BytesKey, codec.CollValue[types.Selection](cdc), NewSelectorsIndex(sb)),
		SelectorTips:              collections.NewMap(sb, types.SelectorTipsPrefix, "delegator_tips", collections.BytesKey, layertypes.LegacyDecValue),
		DisputedDelegationAmounts: collections.NewMap(sb, types.DisputedDelegationAmountsPrefix, "disputed_delegation_amounts", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc)),
		FeePaidFromStake:          collections.NewMap(sb, types.FeePaidFromStakePrefix, "fee_paid_from_stake", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc)),
		Report: collections.NewIndexedMap(
			sb, types.ReporterPrefix, "report",
			collections.PairKeyCodec(collections.BytesKey, collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)), codec.CollValue[types.DelegationsAmounts](cdc), newReportIndexes(sb),
		),
		ReporterPeriodData:       collections.NewMap(sb, types.ReporterPeriodDataPrefix, "reporter_period_data", collections.BytesKey, codec.CollValue[types.PeriodRewardData](cdc)),
		DistributionQueue:        collections.NewMap(sb, types.DistributionQueuePrefix, "distribution_queue", collections.Uint64Key, codec.CollValue[types.DistributionQueueItem](cdc)),
		DistributionQueueCounter: collections.NewItem(sb, types.DistributionQueueCounterPrefix, "distribution_queue_counter", codec.CollValue[types.DistributionQueueCounter](cdc)),
		authority:                authority,
		logger:                   logger,
		accountKeeper:            accountKeeper,
		stakingKeeper:            stakingKeeper,
		bankKeeper:               bankKeeper,
		registryKeeper:           registryKeeper,
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

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetDelegatorTokensAtBlock returns the total amount of tokens a selector had when part of reporting to the nearest given block Number.
func (k Keeper) GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber uint64) (math.Int, error) {
	del, err := k.Selectors.Get(ctx, delegator)
	if err != nil {
		return math.Int{}, err
	}
	rep, err := k.GetDelegationsAmount(ctx, del.Reporter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	delegatorTokens := math.ZeroInt()
	// token origins {selector, validator, amount}
	// loop through token origins and sum up the amount for the selector
	for _, r := range rep.TokenOrigins {
		if bytes.Equal(r.DelegatorAddress, delegator) {
			delegatorTokens = delegatorTokens.Add(r.Amount)
		}
	}
	return delegatorTokens, nil
}

// GetReporterTokensAtBlock returns the total amount of tokens a reporter when reporting to the nearest given block Number.
func (k Keeper) GetReporterTokensAtBlock(ctx context.Context, reporter []byte, blockNumber uint64) (math.Int, error) {
	rep, err := k.GetDelegationsAmount(ctx, reporter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	if rep.Total.IsNil() {
		return math.ZeroInt(), nil
	}
	return rep.Total, nil
}

func (k Keeper) GetDelegationsAmount(ctx context.Context, reporter []byte, blockNumber uint64) (delAmounts types.DelegationsAmounts, err error) {
	start := collections.Join(reporter, uint64(0))
	end := collections.Join(reporter, blockNumber+1)
	pc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	startBuffer := make([]byte, pc.Size(start))
	endBuffer := make([]byte, pc.Size(end))
	_, err = pc.Encode(startBuffer, start)
	if err != nil {
		return delAmounts, err
	}
	_, err = pc.Encode(endBuffer, end)
	if err != nil {
		return delAmounts, err
	}

	iter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuffer, endBuffer, collections.OrderDescending)
	if err != nil {
		return delAmounts, err
	}
	if iter.Valid() {
		key, err := iter.Key()
		if err != nil {
			return delAmounts, err
		}

		rep, err := k.Report.Get(ctx, collections.Join(key.K2(), collections.Join(key.K1().K1(), key.K1().K2())))
		if err != nil {
			return delAmounts, err
		}

		return rep, nil
	}
	return delAmounts, nil
}

// tracks total bonded tokens and sets an expiration of 12 hours from last update
// sets the total BONDED tokens at time of update
func (k Keeper) TrackStakeChange(ctx context.Context) error {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	maxStake, err := k.Tracker.Get(ctx)
	if err != nil {
		return err
	}
	if sdkctx.BlockTime().Before(*maxStake.Expiration) {
		return nil
	}
	// reset expiration
	newExpiration := sdkctx.BlockTime().Add(12 * time.Hour)
	// get current total stake
	total, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}

	maxStake.Expiration = &newExpiration
	maxStake.Amount = total
	return k.Tracker.Set(ctx, maxStake)
}

func CalculateRewardAmount(reporterPower, totalPower uint64, reward math.Int) math.LegacyDec {
	rPower := math.LegacyNewDec(int64(reporterPower))
	tPower := math.LegacyNewDec(int64(totalPower))
	amount := rPower.Quo(tPower).Mul(reward.ToLegacyDec())
	return amount
}

func (k *Keeper) SetOracleKeeper(ok types.OracleKeeper) {
	k.oracleKeeper = ok
}

// gets the block number of the last report for a reporter
func (k Keeper) GetLastReportedAtBlock(ctx context.Context, reporter []byte) (uint64, error) {
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	pc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	start := collections.Join(reporter, uint64(0))
	end := collections.Join(reporter, uint64(currentBlock+1))
	startBuf := make([]byte, pc.Size(start))
	endBuf := make([]byte, pc.Size(end))
	_, _ = pc.Encode(startBuf, start)
	_, _ = pc.Encode(endBuf, end)
	iter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuf, endBuf, collections.OrderDescending)
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	if iter.Valid() {
		key, err := iter.Key()
		if err != nil {
			return 0, err
		}
		blockNumber := key.K1().K2()
		return blockNumber, nil
	}
	return 0, nil
}

// PruneOldReports removes Report entries older than 60 days.
// It iterates the BlockNumber index, checks each block's timestamp via oracle,
// and deletes entries where timestamp < cutoff (60 days ago).
// maxIterations limits how many entries are processed per call.
func (k Keeper) PruneOldReports(ctx context.Context, maxIterations int) error {
	if k.oracleKeeper == nil {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Calculate cutoff timestamp (60 days ago)
	cutoffTimestamp := uint64(sdkCtx.BlockTime().Add(-60 * 24 * time.Hour).UnixMilli())

	// Iterate BlockNumber index in ascending order
	// Index key structure from IterateRaw: Pair[Pair[reporter, blockNumber], queryId]
	pc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	start := collections.Join([]byte{}, uint64(0))
	startBuffer := make([]byte, pc.Size(start))
	_, _ = pc.Encode(startBuffer, start)

	iter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuffer, nil, collections.OrderAscending)
	if err != nil {
		return err
	}
	defer iter.Close()

	keysToDelete := make([]collections.Pair[[]byte, collections.Pair[[]byte, uint64]], 0)
	// Cache: blockNumber -> isOld (true if timestamp < cutoff)
	// because multiple Report entries can exist at the same block height (different reporters/queryIds)
	// we want to avoid redundant oracle calls for the same block height
	checkedBlocks := make(map[uint64]bool)
	iterations := 0

	for ; iter.Valid() && iterations < maxIterations; iter.Next() {
		iterations++

		key, err := iter.Key()
		if err != nil {
			continue
		}

		blockNumber := key.K1().K2()

		// Check cache first to avoid redundant oracle calls
		isOld, found := checkedBlocks[blockNumber]
		if !found {
			// Query oracle for timestamp at this block height
			timestamp, err := k.oracleKeeper.GetTimestampForBlockHeight(ctx, blockNumber)
			if err != nil || timestamp == 0 {
				// Can't determine age, skip this entry
				continue
			}
			isOld = timestamp < cutoffTimestamp
			checkedBlocks[blockNumber] = isOld
		}

		if isOld {
			// Reconstruct primary key: (queryId, (reporter, blockNumber))
			primaryKey := collections.Join(key.K2(), collections.Join(key.K1().K1(), key.K1().K2()))
			keysToDelete = append(keysToDelete, primaryKey)
		}
	}

	// Delete old entries, can't delete during iteration
	for _, key := range keysToDelete {
		if err := k.Report.Remove(ctx, key); err != nil {
			return err
		}
	}

	return nil
}
