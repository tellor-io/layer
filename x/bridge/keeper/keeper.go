package keeper

import (
	"bytes"
	"fmt"

	gomath "math"

	math "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"sort"

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
		k.Logger(ctx).Info("@GetBridgeValidators - bridge validator ", "test", bridgeValset[i].EthereumAddress)
	}

	// Sort the validators
	sort.Slice(bridgeValset, func(i, j int) bool {
		if bridgeValset[i].Power == bridgeValset[j].Power {
			// If power is equal, sort alphabetically
			return bridgeValset[i].EthereumAddress < bridgeValset[j].EthereumAddress
		}
		// Otherwise, sort by power in descending order
		return bridgeValset[i].Power > bridgeValset[j].Power
	})

	return bridgeValset, nil
}

func (k Keeper) GetBridgeValidatorSet(ctx sdk.Context) (*types.BridgeValidatorSet, error) {
	// validators := k.stakingKeeper.GetAllValidators(ctx)
	// bridgeValset := make([]*types.BridgeValidator, len(validators))
	// for i, validator := range validators {
	// 	bridgeValset[i] = &types.BridgeValidator{
	// 		EthereumAddress: DefaultEVMAddress(validator.GetOperator()).String(),
	// 		Power:           uint64(validator.GetConsensusPower(math.NewInt(10))),
	// 	}
	// 	k.Logger(ctx).Info("@GetBridgeValidatorSet - bridge validator ", "test", bridgeValset[i].EthereumAddress)
	// }

	// // Sort the validators
	// sort.Slice(bridgeValset, func(i, j int) bool {
	// 	if bridgeValset[i].Power == bridgeValset[j].Power {
	// 		// If power is equal, sort alphabetically
	// 		return bridgeValset[i].EthereumAddress < bridgeValset[j].EthereumAddress
	// 	}
	// 	// Otherwise, sort by power in descending order
	// 	return bridgeValset[i].Power > bridgeValset[j].Power
	// })

	// use GetBridgeValidators to get the current bridge validator set
	bridgeValset, err := k.GetBridgeValidators(ctx)
	if err != nil {
		return nil, err
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
	if bytes.Equal(k.cdc.MustMarshal(lastSavedBridgeValidators), k.cdc.MustMarshal(currentBridgeValidators)) {
		return true, nil
	} else if k.PowerDiff(ctx, lastSavedBridgeValidators, currentBridgeValidators) < 0.05 {
		k.Logger(ctx).Info("Power diff is less than 5%")
		return false, nil
	} else {
		k.SetBridgeValidators(ctx, currentBridgeValidators)
		k.Logger(ctx).Info("Bridge validator set updated")
		for i, validator := range lastSavedBridgeValidators.BridgeValidatorSet {
			k.Logger(ctx).Info("Last saved bridge validator ", "savedVal", validator.EthereumAddress)
			k.Logger(ctx).Info("i ", "i", i)
		}
		for i, validator := range currentBridgeValidators.BridgeValidatorSet {
			k.Logger(ctx).Info("Current bridge validator ", i, ": ", validator.EthereumAddress+" "+fmt.Sprint(validator.Power))
		}
		return true, nil
	}
}

func (k Keeper) PowerDiff(ctx sdk.Context, b *types.BridgeValidatorSet, c *types.BridgeValidatorSet) float64 {
	powers := map[string]int64{}
	for _, bv := range b.BridgeValidatorSet {
		powers[bv.EthereumAddress] = int64(bv.GetPower())
	}

	for _, bv := range c.BridgeValidatorSet {
		if val, ok := powers[bv.EthereumAddress]; ok {
			powers[bv.EthereumAddress] = val - int64(bv.GetPower())
		} else {
			powers[bv.EthereumAddress] = -int64(bv.GetPower())
		}
	}

	var delta float64
	for _, v := range powers {
		delta += gomath.Abs(float64(v))
	}

	return gomath.Abs(delta / float64(gomath.MaxUint32))
}
