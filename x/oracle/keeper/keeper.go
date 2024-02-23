package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/oracle/types"
)

type (
	Keeper struct {
		cdc            codec.BinaryCodec
		storeService   store.KVStoreService
		Params         collections.Item[types.Params]
		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		registryKeeper types.RegistryKeeper
		reporterKeeper types.ReporterKeeper
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		Schema     collections.Schema
		Commits    collections.Map[collections.Pair[[]byte, []byte], types.CommitReport]                               // key: reporter, queryid
		Tips       *collections.IndexedMap[collections.Pair[[]byte, []byte], math.Int, tipsIndex]                      // key: queryId, tipper
		TotalTips  collections.Item[math.Int]                                                                          // keep track of the total tips
		Aggregates collections.Map[collections.Pair[[]byte, int64], types.Aggregate]                                   // key: queryId, timestamp
		Nonces     collections.Map[[]byte, uint64]                                                                     // key: queryId
		Reports    *collections.IndexedMap[collections.Triple[[]byte, []byte, int64], types.MicroReport, reportsIndex] // key: queryId, reporter, blockHeight
		CycleIndex collections.Item[int64]                                                                             // keep track of the current cycle

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
		Nonces:     collections.NewMap(sb, types.NoncesPrefix, "nonces", collections.BytesKey, collections.Uint64Value),
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
