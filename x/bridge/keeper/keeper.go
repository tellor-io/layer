package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
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

func (k Keeper) GetBridgeValidators(ctx sdk.Context) ([]gethcommon.Address, error) {
	singleVal := k.stakingKeeper.GetValidators(ctx, uint32(1))
	k.Logger(ctx).Info("Single Validator: ", singleVal)
	singleVal2 := k.stakingKeeper.GetAllValidators(ctx)[0]
	k.Logger(ctx).Info("Single Validator 2: ", singleVal2)
	validators := k.stakingKeeper.GetAllValidators(ctx)
	valAddr, _ := sdk.ValAddressFromBech32(validators[0].OperatorAddress)
	k.Logger(ctx).Info("OperatorAddress: ", validators[0].OperatorAddress)
	k.Logger(ctx).Info("ValAddr: ", valAddr)
	k.Logger(ctx).Info("Validators: ", validators)
	// create list of ethereum versions of these validator addresses using DefaultEVMAddress from eth_address.go
	ethAddresses := make([]gethcommon.Address, len(validators))
	for i, validator := range validators {
		ethAddresses[i] = DefaultEVMAddress(validator.GetOperator())
	}

	k.Logger(ctx).Info("Ethereum Addresses: ", ethAddresses)

	return ethAddresses, nil
}
