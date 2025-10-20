package v_6_0_0

import (
	"context"
	"fmt"

	minttypes "github.com/tellor-io/layer/x/mint/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

/*
	Upgrade to v6.0.0 will include:
		- adding ExtraRewardsPool module account to the mint module
		- adding SendExtraRewards function to the mint keeper to send any balance in the ExtraRewardsPool to the TimeBasedRewards account on each BeginBlock

*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak keeper.AccountKeeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))
		if acc := ak.GetModuleAccount(sdkCtx, minttypes.ExtraRewardsPool); acc == nil {
			sdkCtx.Logger().Info("Creating ExtraRewardsPool module account")

			addr := authtypes.NewModuleAddress(minttypes.ExtraRewardsPool)
			extraRewardsPoolMacc := authtypes.NewEmptyModuleAccount(minttypes.ExtraRewardsPool)
			ak.SetModuleAccount(sdkCtx, extraRewardsPoolMacc)

			sdkCtx.Logger().Info("Created ExtraRewardsPool module account", "address", addr.String())
		} else {
			sdkCtx.Logger().Info("ExtraRewardsPool module account already exists")
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
