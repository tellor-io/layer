package keeper

import (
	"bytes"
	"context"
	"errors"
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
		ReportByBlock             *collections.IndexedMap[collections.Triple[[]byte, uint64, []byte], types.DelegationsAmounts, ReportByBlockIndexes]                       // key: reporter AccAddress, blockNumber, queryId
		StakeSnapshots            collections.Map[collections.Triple[[]byte, uint64, uint64], types.DelegationsAmounts]                                                     // key: reporter AccAddress, blockNumber, version
		StakeSnapshotVersion      collections.Sequence                                                                                                                      // numbers each saved stake snapshot, so later ones sort after earlier ones

		// Distribution queue collections
		ReporterPeriodData       collections.Map[[]byte, types.PeriodRewardData]      // key: reporter AccAddress -> current period data
		DistributionQueue        collections.Map[uint64, types.DistributionQueueItem] // key: queue index -> item to distribute
		DistributionQueueCounter collections.Item[types.DistributionQueueCounter]     // tracks head and tail of queue

		LastValSetUpdateHeight collections.Item[uint64]       // block height of last validator set update
		StakeRecalcFlag        collections.Map[[]byte, bool]  // reporters flagged for stake recalculation
		RecalcAtTime           collections.Map[[]byte, int64] // per-reporter earliest lock expiry in seconds requiring recalculation

		// Deferred reporter switches: key (outgoing_reporter, selector) -> target + unlock height.
		OutgoingPendingSwitches collections.Map[collections.Pair[[]byte, []byte], types.PendingSwitchEntry]
		// Incoming index: key (incoming_reporter, selector) -> outgoing_reporter address bytes.
		IncomingPendingSwitchIdx collections.Map[collections.Pair[[]byte, []byte], []byte]
		// One row per reporter: counts + min unlock for O(1) checks in ReporterStake.
		ReporterPendingSwitchHeads collections.Map[[]byte, types.ReporterPendingSwitchHead]
		// Per-dispute selector jail: (selector, dispute_hash_id) -> unix seconds (lock until).
		SelectorDisputeLocks collections.Map[collections.Pair[[]byte, []byte], int64]

		// Tip unlock-to-balance: mid-unbonding tips awaiting maturity payout.
		TipUnlocks     collections.Map[collections.Pair[[]byte, uint64], types.TipUnlockEntry] // (selector, unlock_id)
		TipUnlockQueue collections.Map[collections.Pair[int64, uint64], []byte]                // (completion_unix, unlock_id) -> selector
		TipUnlockID    collections.Sequence

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
		ReportByBlock: collections.NewIndexedMap(
			sb, types.ReportByBlockPrefix, "report_by_block",
			collections.TripleKeyCodec(collections.BytesKey, collections.Uint64Key, collections.BytesKey),
			codec.CollValue[types.DelegationsAmounts](cdc), newReportByBlockIndexes(sb),
		),
		StakeSnapshots: collections.NewMap(
			sb, types.StakeSnapshotPrefix, "stake_snapshots",
			collections.TripleKeyCodec(collections.BytesKey, collections.Uint64Key, collections.Uint64Key),
			codec.CollValue[types.DelegationsAmounts](cdc),
		),
		StakeSnapshotVersion:     collections.NewSequence(sb, types.StakeSnapshotVersionPrefix, "stake_snapshot_version"),
		ReporterPeriodData:       collections.NewMap(sb, types.ReporterPeriodDataPrefix, "reporter_period_data", collections.BytesKey, codec.CollValue[types.PeriodRewardData](cdc)),
		DistributionQueue:        collections.NewMap(sb, types.DistributionQueuePrefix, "distribution_queue", collections.Uint64Key, codec.CollValue[types.DistributionQueueItem](cdc)),
		DistributionQueueCounter: collections.NewItem(sb, types.DistributionQueueCounterPrefix, "distribution_queue_counter", codec.CollValue[types.DistributionQueueCounter](cdc)),
		LastValSetUpdateHeight:   collections.NewItem(sb, types.LastValSetUpdateHeightPrefix, "last_val_set_update_height", collections.Uint64Value),
		StakeRecalcFlag:          collections.NewMap(sb, types.StakeRecalcFlagPrefix, "stake_recalc_flag", collections.BytesKey, collections.BoolValue),
		RecalcAtTime:             collections.NewMap(sb, types.RecalcAtTimePrefix, "recalc_at_time", collections.BytesKey, collections.Int64Value),
		OutgoingPendingSwitches: collections.NewMap(sb,
			types.OutgoingPendingSwitchPrefix,
			"outgoing_pending_switch",
			collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			codec.CollValue[types.PendingSwitchEntry](cdc),
		),
		IncomingPendingSwitchIdx: collections.NewMap(sb,
			types.IncomingPendingSwitchIdxPrefix,
			"incoming_pending_switch_idx",
			collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			collections.BytesValue,
		),
		ReporterPendingSwitchHeads: collections.NewMap(sb,
			types.ReporterPendingSwitchHeadPrefix,
			"reporter_pending_switch_heads",
			collections.BytesKey,
			codec.CollValue[types.ReporterPendingSwitchHead](cdc),
		),
		SelectorDisputeLocks: collections.NewMap(sb,
			types.SelectorDisputeLockPrefix,
			"selector_dispute_locks",
			collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			collections.Int64Value,
		),
		TipUnlocks: collections.NewMap(sb,
			types.TipUnlocksPrefix,
			"tip_unlocks",
			collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key),
			codec.CollValue[types.TipUnlockEntry](cdc),
		),
		TipUnlockQueue: collections.NewMap(sb,
			types.TipUnlockQueuePrefix,
			"tip_unlock_queue",
			collections.PairKeyCodec(collections.Int64Key, collections.Uint64Key),
			collections.BytesValue,
		),
		TipUnlockID:    collections.NewSequence(sb, types.TipUnlockIDPrefix, "tip_unlock_id"),
		authority:      authority,
		logger:         logger,
		accountKeeper:  accountKeeper,
		stakingKeeper:  stakingKeeper,
		bankKeeper:     bankKeeper,
		registryKeeper: registryKeeper,
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

// GetDelegatorTokensAtBlock returns the total amount of tokens a selector had when
// part of reporting for their current Selection.reporter, nearest to blockNumber.
func (k Keeper) GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber uint64) (math.Int, error) {
	del, err := k.Selectors.Get(ctx, delegator)
	if err != nil {
		return math.Int{}, err
	}
	return k.GetDelegatorTokensFromReporterAtBlock(ctx, delegator, del.Reporter, blockNumber)
}

// GetDelegatorTokensFromReporterAtBlock returns the tokens attributed to delegator
// in reporter's nearest stake snapshot at or before blockNumber. Used when a
// selector has switched reporters but must peel from a prior reporter's vote.
func (k Keeper) GetDelegatorTokensFromReporterAtBlock(ctx context.Context, delegator, reporter []byte, blockNumber uint64) (math.Int, error) {
	rep, err := k.GetDelegationsAmount(ctx, reporter, blockNumber)
	if err != nil {
		return math.Int{}, err
	}
	delegatorTokens := math.ZeroInt()
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

// latestStakeSnapshot returns the newest versioned stake snapshot saved at or before a block.
func (k Keeper) latestStakeSnapshot(ctx context.Context, reporter []byte, upToBlock uint64) (types.StakeSnapshotRef, types.DelegationsAmounts, bool, error) {
	rng := new(collections.Range[collections.Triple[[]byte, uint64, uint64]]).
		StartInclusive(collections.Join3(reporter, uint64(0), uint64(0)))
	if upToBlock == ^uint64(0) {
		rng = rng.EndInclusive(collections.Join3(reporter, upToBlock, ^uint64(0)))
	} else {
		rng = rng.EndExclusive(collections.Join3(reporter, upToBlock+1, uint64(0)))
	}
	iter, err := k.StakeSnapshots.Iterate(ctx, rng.Descending())
	if err != nil {
		return types.StakeSnapshotRef{}, types.DelegationsAmounts{}, false, err
	}
	defer iter.Close()
	if !iter.Valid() {
		return types.StakeSnapshotRef{}, types.DelegationsAmounts{}, false, nil
	}
	kv, err := iter.KeyValue()
	if err != nil {
		return types.StakeSnapshotRef{}, types.DelegationsAmounts{}, false, err
	}
	return types.StakeSnapshotRef{Block: kv.Key.K2(), Version: kv.Key.K3()}, kv.Value, true, nil
}

// GetReportStakeSnapshot returns the stake snapshot that backed one specific report.
func (k Keeper) GetReportStakeSnapshot(ctx context.Context, reporter []byte, reportBlock uint64, ref types.StakeSnapshotRef) (types.DelegationsAmounts, error) {
	if ref.IsSet() {
		snap, err := k.StakeSnapshots.Get(ctx, collections.Join3(reporter, ref.Block, ref.Version))
		if err == nil {
			return snap, nil
		}
		if !errors.Is(err, collections.ErrNotFound) {
			return types.DelegationsAmounts{}, err
		}
	}
	// Reports made before versions existed fall back to the nearest snapshot by block.
	return k.GetDelegationsAmount(ctx, reporter, reportBlock)
}

func (k Keeper) GetDelegationsAmount(ctx context.Context, reporter []byte, blockNumber uint64) (delAmounts types.DelegationsAmounts, err error) {
	// The newest versioned snapshot wins; older storage is only read when there is none.
	_, versioned, found, err := k.latestStakeSnapshot(ctx, reporter, blockNumber)
	if err != nil {
		return delAmounts, err
	}
	if found {
		return versioned, nil
	}

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

	// Check new collection first
	newIter, err := k.ReportByBlock.IterateRaw(ctx, startBuffer, endBuffer, collections.OrderDescending)
	if err != nil {
		return delAmounts, err
	}
	defer newIter.Close()
	if newIter.Valid() {
		val, err := newIter.Value()
		if err != nil {
			return delAmounts, err
		}
		return val, nil
	}

	// Fallback to old collection only if new collection had nothing
	oldIter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuffer, endBuffer, collections.OrderDescending)
	if err != nil {
		return delAmounts, err
	}
	defer oldIter.Close()
	if oldIter.Valid() {
		key, err := oldIter.Key()
		if err != nil {
			return delAmounts, err
		}
		return k.Report.Get(ctx, collections.Join(key.K2(), collections.Join(key.K1().K1(), key.K1().K2())))
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

func (k *Keeper) FlagStakeRecalc(ctx context.Context, reporter sdk.AccAddress) error {
	return k.StakeRecalcFlag.Set(ctx, reporter.Bytes(), true)
}

func (k *Keeper) SetOracleKeeper(ok types.OracleKeeper) {
	k.oracleKeeper = ok
}

// gets the block number of the last report for a reporter
func (k Keeper) GetLastReportedAtBlock(ctx context.Context, reporter []byte) (uint64, error) {
	currentBlock := sdk.UnwrapSDKContext(ctx).BlockHeight()
	ref, _, found, err := k.latestStakeSnapshot(ctx, reporter, uint64(currentBlock))
	if err != nil {
		return 0, err
	}
	if found {
		return ref.Block, nil
	}
	pc := collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key)
	start := collections.Join(reporter, uint64(0))
	end := collections.Join(reporter, uint64(currentBlock+1))
	startBuf := make([]byte, pc.Size(start))
	endBuf := make([]byte, pc.Size(end))
	_, _ = pc.Encode(startBuf, start)
	_, _ = pc.Encode(endBuf, end)

	// Check new collection first
	newIter, err := k.ReportByBlock.IterateRaw(ctx, startBuf, endBuf, collections.OrderDescending)
	if err != nil {
		return 0, err
	}
	defer newIter.Close()
	if newIter.Valid() {
		key, err := newIter.Key()
		if err != nil {
			return 0, err
		}
		return key.K2(), nil
	}

	// Fallback to old collection only if new collection had nothing
	oldIter, err := k.Report.Indexes.BlockNumber.IterateRaw(ctx, startBuf, endBuf, collections.OrderDescending)
	if err != nil {
		return 0, err
	}
	defer oldIter.Close()
	if oldIter.Valid() {
		key, err := oldIter.Key()
		if err != nil {
			return 0, err
		}
		return key.K1().K2(), nil
	}
	return 0, nil
}

// PruneOldReports removes Report entries older than 30 days.
// It finds the cutoff block height with by calling the oracle using
// ETH/USD aggregates, then iterates and deletes entries below that height.
// It keeps at least one entry per reporter so dispute voting always has
// a snapshot to read historical power from.
func (k Keeper) PruneOldReports(ctx context.Context, maxBatchSize int) error {
	cutoffBlock, ok := k.reportCutoffBlock(ctx)
	if !ok {
		return nil
	}

	totalDeleted := 0

	// Prune old collection first
	type oldKey = collections.Pair[[]byte, collections.Pair[[]byte, uint64]]
	var oldToDelete []oldKey
	// Track the most recent old entry per reporter. Keep it so dispute
	// voting always has at least one snapshot to read power from.
	oldLatest := make(map[string]uint64)
	oldScanned := 0

	oldIter, err := k.Report.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer oldIter.Close()
	for ; oldIter.Valid() && oldScanned < maxBatchSize && len(oldToDelete) < maxBatchSize; oldIter.Next() {
		oldScanned++
		pk, err := oldIter.Key()
		if err != nil {
			return err
		}
		if pk.K2().K2() < cutoffBlock {
			oldToDelete = append(oldToDelete, pk)
			reporter := string(pk.K2().K1())
			if block := pk.K2().K2(); block > oldLatest[reporter] {
				oldLatest[reporter] = block
			}
		}
	}

	for _, pk := range oldToDelete {
		if pk.K2().K2() == oldLatest[string(pk.K2().K1())] {
			continue
		}
		if err := k.Report.Remove(ctx, pk); err != nil {
			return err
		}
		totalDeleted++
	}

	// Prune new collection (ReportByBlock), iterate from lowest blockNumber, break at cutoff
	remaining := maxBatchSize - totalDeleted
	if remaining > 0 {
		type newKey = collections.Triple[[]byte, uint64, []byte]
		var newToDelete []newKey
		// Same retention logic for the new collection
		newLatest := make(map[string]uint64)
		newIter, err := k.ReportByBlock.Indexes.BlockNumber.Iterate(ctx, nil)
		if err != nil {
			return err
		}
		defer newIter.Close()
		for ; newIter.Valid() && len(newToDelete) < remaining; newIter.Next() {
			pk, err := newIter.PrimaryKey()
			if err != nil {
				return err
			}

			if pk.K2() >= cutoffBlock {
				break
			}
			newToDelete = append(newToDelete, pk)
			reporter := string(pk.K1())
			if block := pk.K2(); block > newLatest[reporter] {
				newLatest[reporter] = block
			}
		}

		for _, pk := range newToDelete {
			if pk.K2() == newLatest[string(pk.K1())] {
				continue
			}
			if err := k.ReportByBlock.Remove(ctx, pk); err != nil {
				return err
			}
			totalDeleted++
		}
	}

	return nil
}

// MaxStakeSnapshotCleanup is how many old snapshots a reporter clears each time it saves a new one.
const MaxStakeSnapshotCleanup = 10

// reportCutoffBlock returns the block from 30 days ago, or false when it can't be worked out yet.
func (k Keeper) reportCutoffBlock(ctx context.Context) (uint64, bool) {
	if k.oracleKeeper == nil {
		return 0, false
	}
	cutoff := sdk.UnwrapSDKContext(ctx).BlockTime().Add(-30 * 24 * time.Hour)
	cutoffBlock, err := k.oracleKeeper.GetBlockHeightFromTimestamp(ctx, cutoff)
	if err != nil {
		return 0, false
	}
	return cutoffBlock, true
}

// cleanupOldStakeSnapshots removes a few of a reporter's snapshots older than 30 days, keeping its newest one from before then.
func (k Keeper) cleanupOldStakeSnapshots(ctx context.Context, reporter []byte) error {
	cutoffBlock, ok := k.reportCutoffBlock(ctx)
	if !ok {
		return nil
	}
	old, err := k.oldStakeSnapshotKeys(ctx, reporter, cutoffBlock, MaxStakeSnapshotCleanup+1)
	if err != nil {
		return err
	}
	// The last one found stays: it is either the newest before the cutoff or is left for next time.
	for i := 0; i+1 < len(old); i++ {
		if err := k.StakeSnapshots.Remove(ctx, old[i]); err != nil {
			return err
		}
	}
	return nil
}

// oldStakeSnapshotKeys lists up to limit of a reporter's snapshots saved before the cutoff, oldest first.
func (k Keeper) oldStakeSnapshotKeys(ctx context.Context, reporter []byte, cutoffBlock uint64, limit int) ([]collections.Triple[[]byte, uint64, uint64], error) {
	iter, err := k.StakeSnapshots.Iterate(ctx, collections.NewPrefixedTripleRange[[]byte, uint64, uint64](reporter))
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	keys := make([]collections.Triple[[]byte, uint64, uint64], 0, limit)
	for ; iter.Valid() && len(keys) < limit; iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, err
		}
		if key.K2() >= cutoffBlock {
			break
		}
		keys = append(keys, key)
	}
	return keys, nil
}
