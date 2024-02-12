package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/reporter/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		Params       collections.Item[types.Params]
		Reporters    collections.Map[sdk.AccAddress, types.OracleReporter]
		Delegators   collections.Map[sdk.AccAddress, types.Delegation]
		TokenOrigin  collections.Map[collections.Pair[sdk.AccAddress, sdk.ValAddress], types.TokenOrigins]
		Schema       collections.Schema
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		stakingKeeper types.StakingKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	stakingKeeper types.StakingKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Reporters:     collections.NewMap(sb, types.ReportersKey, "reporters_by_reporter", sdk.AccAddressKey, codec.CollValue[types.OracleReporter](cdc)),
		Delegators:    collections.NewMap(sb, types.DelegatorsKey, "delegation_by_delegator", sdk.AccAddressKey, codec.CollValue[types.Delegation](cdc)),
		TokenOrigin:   collections.NewMap(sb, types.TokenOriginsKey, "token_origins_by_reporter", collections.PairKeyCodec(sdk.AccAddressKey, sdk.ValAddressKey), codec.CollValue[types.TokenOrigins](cdc)),
		authority:     authority,
		logger:        logger,
		stakingKeeper: stakingKeeper,
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
