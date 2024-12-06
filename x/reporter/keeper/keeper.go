package keeper

import (
	"bytes"
	"context"
	"fmt"
	"time"

	layertypes "github.com/tellor-io/layer/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
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
		Tip                       collections.Map[uint64, oracletypes.Reward]                                                                                               // key: QueryMeta.Id
		Tbr                       collections.Map[uint64, oracletypes.Reward]                                                                                               // key: blockNumer
		ClaimStatus               collections.Map[collections.Pair[[]byte, uint64], bool]

		Schema collections.Schema
		logger log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

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
		Tip:            collections.NewMap(sb, types.TipPrefix, "tips", collections.Uint64Key, codec.CollValue[oracletypes.Reward](cdc)),
		Tbr:            collections.NewMap(sb, types.TbrPrefix, "tbr", collections.Uint64Key, codec.CollValue[oracletypes.Reward](cdc)),
		ClaimStatus:    collections.NewMap(sb, types.ClaimStatusPrefix, "claim_status", collections.PairKeyCodec(collections.BytesKey, collections.Uint64Key), collections.BoolValue),
		authority:      authority,
		logger:         logger,
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

func (k *Keeper) SetOracleKeeper(o types.OracleKeeper) {
	k.oracleKeeper = o
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

func (k Keeper) AddTip(ctx context.Context, metaId uint64, tip oracletypes.Reward) error {
	return k.Tip.Set(ctx, metaId, tip)
}

func (k Keeper) AddTbr(ctx context.Context, metaId uint64, tbr oracletypes.Reward) error {
	return k.Tbr.Set(ctx, metaId, tbr)
}

func (k Keeper) RewardByReporter(ctx context.Context, selAddr, repAddr sdk.AccAddress, metaId uint64, queryId []byte) (math.LegacyDec, error) {
	// ensure the selector hasn't claimed tips already
	claimed, err := k.ClaimStatus.Has(ctx, collections.Join(selAddr.Bytes(), metaId))
	if err != nil {
		return math.LegacyDec{}, err
	}
	if claimed {
		return math.LegacyZeroDec(), nil
	}

	report, err := k.oracleKeeper.MicroReport(ctx, collections.Join3(queryId, repAddr.Bytes(), metaId))
	if err != nil {
		return math.LegacyDec{}, err
	}

	reporterTipAmount := math.LegacyZeroDec()
	reporterTbrAmount := math.LegacyZeroDec()

	// a tip object should always exist for a legit metaId set during aggregation even if amount is 0
	tipobj, err := k.Tip.Get(ctx, metaId)
	if err != nil {
		return math.LegacyDec{}, err
	}

	tip := tipobj.Amount
	if tip.IsPositive() {
		reporterTipAmount = CalculateRewardAmount(
			report.Power,
			tipobj.TotalPower,
			tip.TruncateInt(),
		)
	}

	if tipobj.CycleList {
		// if query is part of cyclist then tbr amount and total power should've been set
		tbrobj, err := k.Tbr.Get(ctx, tipobj.BlockHeight)
		if err != nil {
			return math.LegacyDec{}, err
		}
		tbr := tbrobj.Amount
		if tbr.IsPositive() {
			reporterTbrAmount = CalculateRewardAmount(
				report.Power,
				tbrobj.TotalPower,
				tbr.TruncateInt(),
			)
		}
	}

	reporter, err := k.Reporters.Get(ctx, repAddr)
	if err != nil {
		return math.LegacyDec{}, err
	}
	reporterAmount := reporterTipAmount.Add(reporterTbrAmount)

	if reporterAmount.IsZero() {
		return reporterAmount, nil
	}
	// selector's commission = reporter's commission rate * reward
	commission := reporterAmount.Mul(reporter.CommissionRate)
	// Calculate net reward
	netReward := reporterAmount.Sub(commission)

	selectors, err := k.Report.Get(ctx, collections.Join(queryId, collections.Join(repAddr.Bytes(), report.BlockNumber)))
	if err != nil {
		return math.LegacyDec{}, err
	}
	selectorsTotalDec := selectors.Total.ToLegacyDec()
	var selectorPortion types.TokenOriginInfo
	for _, selector := range selectors.TokenOrigins {
		if bytes.Equal(selector.DelegatorAddress, selAddr.Bytes()) {
			selectorPortion = *selector
			break
		}
	}

	// check if selector found
	if selectorPortion.Amount.IsNil() {
		return math.LegacyZeroDec(), nil
	}
	selAmountDec := selectorPortion.Amount.ToLegacyDec()
	selectorShare := netReward.Mul(selAmountDec).Quo(selectorsTotalDec)

	if selAddr.Equals(repAddr) {
		selectorShare = selectorShare.Add(commission)
	}
	fmt.Println("selector share", selectorShare)
	return selectorShare, nil
}

func CalculateRewardAmount(reporterPower, totalPower uint64, reward math.Int) math.LegacyDec {
	rPower := math.LegacyNewDec(int64(reporterPower))
	tPower := math.LegacyNewDec(int64(totalPower))
	amount := rPower.Quo(tPower).Mul(reward.ToLegacyDec())
	return amount
}
