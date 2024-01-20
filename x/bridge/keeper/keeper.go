package keeper

import (
	"fmt"

	math "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	"github.com/tellor-io/layer/x/bridge/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		stakingKeeper  types.StakingKeeper
		slashingKeeper types.SlashingKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	stakingKeeper types.StakingKeeper,
	slashingKeeper types.SlashingKeeper,
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

		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetBridgeValidators(ctx sdk.Context) ([]*types.BridgeValidator, error) {
	validators := k.stakingKeeper.GetAllValidators(ctx)

	bridgeValset := make([]*types.BridgeValidator, len(validators))

	// ethAddresses := make([]gethcommon.Address, len(validators))
	for i, validator := range validators {
		// ethAddresses[i] = DefaultEVMAddress(validator.GetOperator())
		// k.Logger(ctx).Info("building eth addrs", i)
		bridgeValset[i] = &types.BridgeValidator{
			EthereumAddress: DefaultEVMAddress(validator.GetOperator()).String(),
			Power:           uint64(validator.GetConsensusPower(math.NewInt(10))),
		}
	}

	return bridgeValset, nil
}

func (k Keeper) GetBridgeValidatorSet(ctx sdk.Context) (*types.BridgeValidatorSet, error) {
	validators := k.stakingKeeper.GetAllValidators(ctx)
	bridgeValset := make([]*types.BridgeValidator, len(validators))
	for i, validator := range validators {
		bridgeValset[i] = &types.BridgeValidator{
			EthereumAddress: DefaultEVMAddress(validator.GetOperator()).String(),
			Power:           uint64(validator.GetConsensusPower(math.NewInt(10))),
		}
	}
	return &types.BridgeValidatorSet{BridgeValidatorSet: bridgeValset}, nil
}

func (k Keeper) SetBridgeValidators(ctx sdk.Context, bridgeValidators *types.BridgeValidatorSet) {
	store := k.BridgeValidatorsStore(ctx) // You need to create this method
	bz := k.cdc.MustMarshal(bridgeValidators)
	store.Set([]byte("BridgeValidators"), bz)
}

func (k Keeper) BridgeValidatorsStore(ctx sdk.Context) storetypes.KVStore {
	return prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix("BridgeValidatorsKey"))
}

// function for loading last saved bridge validator set
func (k Keeper) GetLastSavedBridgeValidators(ctx sdk.Context) (*types.BridgeValidatorSet, error) {
	store := k.BridgeValidatorsStore(ctx)
	bz := store.Get([]byte("BridgeValidators"))
	if bz == nil {
		return nil, fmt.Errorf("no bridge validator set found")
	}
	var bridgeValidators types.BridgeValidatorSet
	k.cdc.MustUnmarshal(bz, &bridgeValidators)
	return &bridgeValidators, nil
}

// function for loading last saved bridge validator set and comparing it to current set
func (k Keeper) CompareBridgeValidators(ctx sdk.Context) (bool, error) {
	currentBridgeValidators, err := k.GetBridgeValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Info("No current bridge validator set found")
		return false, err
	}
	lastSavedBridgeValidators, err := k.GetLastSavedBridgeValidators(ctx)
	if err != nil {
		k.Logger(ctx).Info("No saved bridge validator set found")
		k.SetBridgeValidators(ctx, currentBridgeValidators)
		return false, err
	}
	if lastSavedBridgeValidators == currentBridgeValidators {
		return true, nil
	} else {
		k.SetBridgeValidators(ctx, currentBridgeValidators)
		k.Logger(ctx).Info("Bridge validator set updated")
		return true, nil
	}
}
