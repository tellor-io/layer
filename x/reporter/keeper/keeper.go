package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

type (
	Keeper struct {
		cdc                            codec.BinaryCodec
		storeService                   store.KVStoreService
		Params                         collections.Item[types.Params]
		Reporters                      collections.Map[sdk.AccAddress, types.OracleReporter]
		Delegators                     *collections.IndexedMap[sdk.AccAddress, types.Delegation, ReporterDelegatorsIndex]
		TokenOrigin                    collections.Map[collections.Pair[sdk.AccAddress, sdk.ValAddress], math.Int]
		ReportersAccumulatedCommission collections.Map[sdk.ValAddress, types.ReporterAccumulatedCommission]
		ReporterOutstandingRewards     collections.Map[sdk.ValAddress, types.ReporterOutstandingRewards]
		ReporterCurrentRewards         collections.Map[sdk.ValAddress, types.ReporterCurrentRewards]
		DelegatorStartingInfo          collections.Map[collections.Pair[sdk.ValAddress, sdk.AccAddress], types.DelegatorStartingInfo]
		ReporterHistoricalRewards      collections.Map[collections.Pair[sdk.ValAddress, uint64], types.ReporterHistoricalRewards]
		ReporterDisputeEvents          collections.Map[collections.Triple[sdk.ValAddress, uint64, uint64], types.ReporterDisputeEvent]
		TokenOriginSnapshot            collections.Map[collections.Pair[sdk.AccAddress, int64], types.DelegationsPreUpdate]

		Schema collections.Schema
		logger log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		stakingKeeper types.StakingKeeper
		bankKeeper    types.BankKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	stakingKeeper types.StakingKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params:                         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Reporters:                      collections.NewMap(sb, types.ReportersKey, "reporters_by_reporter", sdk.AccAddressKey, codec.CollValue[types.OracleReporter](cdc)),
		Delegators:                     collections.NewIndexedMap(sb, types.DelegatorsKey, "delegations_by_delegator", sdk.AccAddressKey, codec.CollValue[types.Delegation](cdc), NewDelegatorsIndex(sb)),
		TokenOrigin:                    collections.NewMap(sb, types.TokenOriginsKey, "token_origins_by_delegator_validator", collections.PairKeyCodec(sdk.AccAddressKey, sdk.ValAddressKey), sdk.IntValue),
		ReportersAccumulatedCommission: collections.NewMap(sb, types.ReporterAccumulatedCommissionPrefix, "reporters_accumulated_commission", sdk.ValAddressKey, codec.CollValue[types.ReporterAccumulatedCommission](cdc)),
		ReporterOutstandingRewards:     collections.NewMap(sb, types.ReporterOutstandingRewardsPrefix, "reporter_outstanding_rewards", sdk.ValAddressKey, codec.CollValue[types.ReporterOutstandingRewards](cdc)),
		ReporterCurrentRewards:         collections.NewMap(sb, types.ReporterCurrentRewardsPrefix, "reporters_current_rewards", sdk.ValAddressKey, codec.CollValue[types.ReporterCurrentRewards](cdc)),
		DelegatorStartingInfo:          collections.NewMap(sb, types.DelegatorStartingInfoPrefix, "delegators_starting_info", collections.PairKeyCodec(sdk.ValAddressKey, sdk.AccAddressKey), codec.CollValue[types.DelegatorStartingInfo](cdc)),
		ReporterHistoricalRewards:      collections.NewMap(sb, types.ReporterHistoricalRewardsPrefix, "reporter_historical_rewards", collections.PairKeyCodec(sdk.ValAddressKey, collections.Uint64Key), codec.CollValue[types.ReporterHistoricalRewards](cdc)),
		ReporterDisputeEvents:          collections.NewMap(sb, types.ReporterDisputeEventPrefix, "reporter_dispute_events", collections.TripleKeyCodec(sdk.ValAddressKey, collections.Uint64Key, collections.Uint64Key), codec.CollValue[types.ReporterDisputeEvent](cdc)),
		TokenOriginSnapshot:            collections.NewMap(sb, types.TokenOriginSnapshotPrefix, "token_origin_snapshot", collections.PairKeyCodec(sdk.AccAddressKey, collections.Int64Key), codec.CollValue[types.DelegationsPreUpdate](cdc)),
		authority:                      authority,
		logger:                         logger,
		stakingKeeper:                  stakingKeeper,
		bankKeeper:                     bankKeeper,
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
