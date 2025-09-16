package v5

import (
	"context"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/tellor-io/layer/x/bridge/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateStore(ctx context.Context, keeper *keeper.Keeper, cdc codec.BinaryCodec) error {
	params, err := keeper.Params.Get(ctx)
	if err != nil {
		return err
	}
	params.MainnetChainId = "tellor-1"
	err = keeper.Params.Set(ctx, params)
	if err != nil {
		return err
	}
	keeper.SetValsetCheckpointDomainSeparator(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if strings.EqualFold(sdkCtx.ChainID(), "layertest-4") {
		currentValidatorSetEVMCompatible, err := keeper.GetCurrentValidatorSetEVMCompatible(ctx)
		if err != nil {
			keeper.Logger(ctx).Info("No current validator set found")
			return err
		}

		err = keeper.BridgeValset.Set(ctx, *currentValidatorSetEVMCompatible)
		if err != nil {
			keeper.Logger(ctx).Info("Error setting bridge validator set: ", "error", err)
			return err
		}
		error := keeper.SetBridgeValidatorParams(ctx, currentValidatorSetEVMCompatible)
		if error != nil {
			keeper.Logger(ctx).Info("Error setting bridge validator params: ", "error", error)
			return error
		}
	}

	return nil
}
