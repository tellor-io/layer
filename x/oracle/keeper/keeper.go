package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/oracle/types"
)

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeKey       storetypes.StoreKey
		memKey         storetypes.StoreKey
		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		distrKeeper    types.DistrKeeper
		stakingKeeper  types.StakingKeeper
		registryKeeper types.RegistryKeeper
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		Schema     collections.Schema
		Commits    collections.Map[collections.Pair[[]byte, []byte], types.CommitReport]                               // key: reporter, queryid
		Tips       *collections.IndexedMap[collections.Pair[[]byte, []byte], math.Int, tipsIndex]                      // key: queryId, tipper
		TotalTips  collections.Item[math.Int]                                                                          // keep track of the total tips
		Aggregates collections.Map[collections.Pair[[]byte, int64], types.Aggregate]                                   // key: queryId, timestamp
		Nonces     collections.Map[[]byte, int64]                                                                      // key: queryId
		Reports    *collections.IndexedMap[collections.Triple[[]byte, []byte, int64], types.MicroReport, reportsIndex] // key: queryId, reporter, blockHeight
		CycleIndex collections.Item[int64]                                                                             // keep track of the current cycle

		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distrKeeper types.DistrKeeper,
	stakingKeeper types.StakingKeeper,
	registryKeeper types.RegistryKeeper,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	if storeKey == nil {
		panic("storeKey cannot be nil")
	}

	if memKey == nil {
		panic("memKey cannot be nil")
	}

	storeService := runtime.NewKVStoreService(storeKey.(*storetypes.KVStoreKey))
	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,

		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
		stakingKeeper:  stakingKeeper,
		registryKeeper: registryKeeper,
		authority:      authority,

		Commits: collections.NewMap(sb, types.CommitsPrefix, "commits", collections.PairKeyCodec(collections.BytesKey, collections.BytesKey), codec.CollValue[types.CommitReport](cdc)),
		Tips: collections.NewIndexedMap(sb,
			types.TipsPrefix,
			"tips",
			collections.PairKeyCodec(collections.BytesKey, collections.BytesKey),
			sdk.IntValue,
			NewTipsIndex(sb),
		),
		TotalTips:  collections.NewItem(sb, types.TotalTipsPrefix, "total_tips", sdk.IntValue),
		Aggregates: collections.NewMap(sb, types.AggregatesPrefix, "aggregates", collections.PairKeyCodec(collections.BytesKey, collections.Int64Key), codec.CollValue[types.Aggregate](cdc)),
		Nonces:     collections.NewMap(sb, types.NoncesPrefix, "nonces", collections.BytesKey, collections.Int64Value),
		Reports: collections.NewIndexedMap(sb,
			types.ReportsPrefix,
			"reports",
			collections.TripleKeyCodec(collections.BytesKey, collections.BytesKey, collections.Int64Key),
			codec.CollValue[types.MicroReport](cdc),
			NewReportsIndex(sb),
		),
		CycleIndex: collections.NewItem(sb, types.CycleIndexPrefix, "cycle_index", collections.Int64Value),
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

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func HashQueryData(queryData []byte) []byte {
	return crypto.Keccak256(queryData)
}
