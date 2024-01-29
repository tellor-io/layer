package keeper

import (
	"bytes"
	"context"
	"fmt"

	gomath "math"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	math "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"sort"

	"github.com/tellor-io/layer/x/bridge/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService storetypes.KVStoreService

		Schema       collections.Schema
		Params       collections.Item[types.Params]
		BridgeValset collections.Item[types.BridgeValidatorSet]

		stakingKeeper  types.StakingKeeper
		slashingKeeper types.SlashingKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,

	stakingKeeper types.StakingKeeper,
	slashingKeeper types.SlashingKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:            cdc,
		storeService:   storeService,
		Params:         collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		BridgeValset:   collections.NewItem(sb, types.BridgeValsetKey, "bridge_valset", codec.CollValue[types.BridgeValidatorSet](cdc)),
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetBridgeValidators(ctx sdk.Context) ([]*types.BridgeValidator, error) {
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}

	bridgeValset := make([]*types.BridgeValidator, len(validators))

	// ethAddresses := make([]gethcommon.Address, len(validators))
	for i, validator := range validators {
		// ethAddresses[i] = DefaultEVMAddress(validator.GetOperator())
		// k.Logger(ctx).Info("building eth addrs", i)
		valAddress, err := sdk.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			return nil, err
		}
		bridgeValset[i] = &types.BridgeValidator{
			EthereumAddress: DefaultEVMAddress(valAddress).String(),
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

// function for loading last saved bridge validator set and comparing it to current set
func (k Keeper) CompareBridgeValidators(ctx sdk.Context) (bool, error) {
	// TODO: double-check this, as it's currently getting the stored valset
	currentBridgeValidators, err := k.GetBridgeValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Info("No current bridge validator set found")
		return false, err
	}
	lastSavedBridgeValidators, err := k.BridgeValset.Get(ctx)
	if err != nil {
		k.Logger(ctx).Info("No saved bridge validator set found")
		k.BridgeValset.Set(ctx, *currentBridgeValidators)
		return false, err
	}
	if bytes.Equal(k.cdc.MustMarshal(&lastSavedBridgeValidators), k.cdc.MustMarshal(currentBridgeValidators)) {
		return true, nil
	} else if k.PowerDiff(ctx, lastSavedBridgeValidators, *currentBridgeValidators) < 0.05 {
		k.Logger(ctx).Info("Power diff is less than 5%")
		return false, nil
	} else {
		err := k.BridgeValset.Set(ctx, *currentBridgeValidators)
		if err != nil {
			return false, err
		}
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

func (k Keeper) PowerDiff(ctx sdk.Context, b types.BridgeValidatorSet, c types.BridgeValidatorSet) float64 {
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
