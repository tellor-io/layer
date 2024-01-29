package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tellor-io/layer/x/registry/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) SetGenesisSpec(ctx sdk.Context) {
	var dataSpec types.DataSpec
	dataSpec.DocumentHash = ""
	dataSpec.ValueType = "uint256"
	dataSpec.AggregationMethod = "weighted-median"
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecRegistryKey))
	store.Set([]byte("SpotPrice"), k.cdc.MustMarshal(&dataSpec))

}

func (k Keeper) GetGenesisSpec(ctx sdk.Context) types.DataSpec {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecRegistryKey))
	spec := store.Get([]byte("SpotPrice"))
	var dataSpec types.DataSpec
	k.cdc.Unmarshal(spec, &dataSpec)
	return dataSpec
}

func (k Keeper) SetGenesisQuery(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueryRegistryKey))
	ethQueryData := SpotQueryData("eth", "usd")
	store.Set(crypto.Keccak256(ethQueryData), ethQueryData)
	btcQueryData := SpotQueryData("btc", "usd")
	store.Set(crypto.Keccak256(btcQueryData), btcQueryData)
	trbQueryData := SpotQueryData("trb", "usd")
	store.Set(crypto.Keccak256(trbQueryData), trbQueryData)
}

func SpotQueryData(symbolA, symbolB string) []byte {
	encodedData, _ := EncodeArguments([]string{"string", "string"}, []string{symbolA, symbolB})

	queryData, _ := EncodeArguments([]string{"string", "bytes"}, []string{"SpotPrice", string(encodedData)})

	return queryData
}

func (k Keeper) GetGenesisQuery(ctx sdk.Context) (string, string, string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.QueryRegistryKey))
	trbQueryData := SpotQueryData("trb", "usd")
	bzTRB := store.Get(crypto.Keccak256(trbQueryData))
	trbHexData := (bytes.HexBytes(bzTRB).String())
	btcQueryData := SpotQueryData("btc", "usd")
	bzBTC := store.Get(crypto.Keccak256(btcQueryData))
	btcHexData := (bytes.HexBytes(bzBTC).String())
	ethQueryData := SpotQueryData("eth", "usd")
	bzETH := store.Get(crypto.Keccak256(ethQueryData))
	ethHexData := (bytes.HexBytes(bzETH).String())

	return trbHexData, btcHexData, ethHexData
}
