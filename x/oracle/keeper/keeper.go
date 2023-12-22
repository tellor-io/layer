package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tellor-io/layer/x/oracle/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		memKey   storetypes.StoreKey

		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		distrKeeper    types.DistrKeeper
		stakingKeeper  types.StakingKeeper
		registryKeeper types.RegistryKeeper
		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
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
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		memKey:   memKey,

		accountKeeper:  accountKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
		stakingKeeper:  stakingKeeper,
		registryKeeper: registryKeeper,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) CommitStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportStoreKey))
}

func (k Keeper) AggregateStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AggregateStoreKey))
}

func (k Keeper) ReporterStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReporterStoreKey))
}

func (k Keeper) ReportsStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
}

func (k Keeper) TipStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.TipStoreKey))
}

func HashQueryData(queryData []byte) []byte {
	return crypto.Keccak256(queryData)
}
