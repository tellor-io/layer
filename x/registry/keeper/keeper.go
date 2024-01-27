package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	// "cosmossdk.io/store/prefix"
	// "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/codec"
	// "github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	// "github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/x/registry/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		// Params key: ParamsKeyPrefix | value: Params
		Params       collections.Item[types.Params]
		SpecRegistry collections.Map[string, types.DataSpec]
		Schema       collections.Schema

		authority string
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	authority string,

) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		storeService: storeService,

		Params:       collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		SpecRegistry: collections.NewMap(sb, types.SpecRegistryKey, "specRegistry", collections.StringKey, codec.CollValue[types.DataSpec](cdc)),
		authority:    authority,
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

// TODO: remove query registration
// func (k Keeper) SetGenesisQuery(ctx sdk.Context) {
// 	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
// 	store := prefix.NewStore(storeAdapter, types.QueryRegistryKey)
// 	ethQueryData := SpotQueryData("eth", "usd")
// 	store.Set(crypto.Keccak256(ethQueryData), ethQueryData)
// 	btcQueryData := SpotQueryData("btc", "usd")
// 	store.Set(crypto.Keccak256(btcQueryData), btcQueryData)
// 	trbQueryData := SpotQueryData("trb", "usd")
// 	store.Set(crypto.Keccak256(trbQueryData), trbQueryData)
// }

// func SpotQueryData(symbolA, symbolB string) []byte {

// 	encodedData, _ := EncodeArguments([]string{"string", "string"}, []string{symbolA, symbolB})

// 	queryData, _ := EncodeArguments([]string{"string", "bytes"}, []string{"SpotPrice", string(encodedData)})

// 	return queryData
// }

// func (k Keeper) GetGenesisQuery(ctx sdk.Context) (string, string, string) {
// 	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
// 	store := prefix.NewStore(storeAdapter, types.QueryRegistryKey)
// 	trbQueryData := SpotQueryData("trb", "usd")
// 	bzTRB := store.Get(crypto.Keccak256(trbQueryData))
// 	trbHexData := (bytes.HexBytes(bzTRB).String())
// 	btcQueryData := SpotQueryData("btc", "usd")
// 	bzBTC := store.Get(crypto.Keccak256(btcQueryData))
// 	btcHexData := (bytes.HexBytes(bzBTC).String())
// 	ethQueryData := SpotQueryData("eth", "usd")
// 	bzETH := store.Get(crypto.Keccak256(ethQueryData))
// 	ethHexData := (bytes.HexBytes(bzETH).String())

// 	return trbHexData, btcHexData, ethHexData
// }
