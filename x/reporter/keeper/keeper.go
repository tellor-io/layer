package keeper

import (
	"bytes"
	"context"
	"fmt"
	"time"

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
		Reporters                 collections.Map[[]byte, types.OracleReporter]
		SelectorTips              collections.Map[[]byte, math.Int]
		Selectors                 *collections.IndexedMap[[]byte, types.Selection, ReporterSelectorsIndex]
		DisputedDelegationAmounts collections.Map[[]byte, types.DelegationsAmounts]
		FeePaidFromStake          collections.Map[[]byte, types.DelegationsAmounts]
		Report                    collections.Map[collections.Pair[[]byte, int64], types.DelegationsAmounts]

		Schema collections.Schema
		logger log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		stakingKeeper  types.StakingKeeper
		bankKeeper     types.BankKeeper
		registryKeeper types.RegistryKeeper
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
		SelectorTips:              collections.NewMap(sb, types.SelectorTipsPrefix, "delegator_tips", collections.BytesKey, sdk.IntValue),
		DisputedDelegationAmounts: collections.NewMap(sb, types.DisputedDelegationAmountsPrefix, "disputed_delegation_amounts", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc)),
		FeePaidFromStake:          collections.NewMap(sb, types.FeePaidFromStakePrefix, "fee_paid_from_stake", collections.BytesKey, codec.CollValue[types.DelegationsAmounts](cdc)),
		Report:                    collections.NewMap(sb, types.ReporterPrefix, "report", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), codec.CollValue[types.DelegationsAmounts](cdc)),
		authority:                 authority,
		logger:                    logger,
		stakingKeeper:             stakingKeeper,
		bankKeeper:                bankKeeper,
		registryKeeper:            registryKeeper,
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

func (k Keeper) GetDelegatorTokensAtBlock(ctx context.Context, delegator []byte, blockNumber int64) (math.Int, error) {
	del, err := k.Selectors.Get(ctx, delegator)
	if err != nil {
		return math.Int{}, err
	}
	rng := collections.NewPrefixedPairRange[[]byte, int64](del.Reporter).EndInclusive(blockNumber).Descending()
	rep := types.DelegationsAmounts{}
	err = k.Report.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.DelegationsAmounts) (bool, error) {
		rep = value
		return true, nil
	})
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

func (k Keeper) GetReporterTokensAtBlock(ctx context.Context, reporter []byte, blockNumber int64) (math.Int, error) {
	rng := collections.NewPrefixedPairRange[[]byte, int64](reporter).EndInclusive(blockNumber).Descending()
	total := math.ZeroInt()
	err := k.Report.Walk(ctx, rng, func(key collections.Pair[[]byte, int64], value types.DelegationsAmounts) (bool, error) {
		total = value.Total
		return true, nil
	})
	if err != nil {
		return math.Int{}, err
	}
	return total, nil
}

func (k Keeper) TrackStakeChange(ctx context.Context) error {
	sdkctx := sdk.UnwrapSDKContext(ctx)
	maxStake, err := k.Tracker.Get(ctx)
	if err != nil {
		return err
	}
	if sdkctx.HeaderInfo().Time.Before(*maxStake.Expiration) {
		return nil
	}
	// reset expiration
	newExpiration := sdkctx.HeaderInfo().Time.Add(12 * time.Hour)
	// get current total stake
	total, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}

	maxStake.Expiration = &newExpiration
	maxStake.Amount = total
	return k.Tracker.Set(ctx, maxStake)
}
