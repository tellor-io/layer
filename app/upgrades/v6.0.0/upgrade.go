package v_6_0_0

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
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

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
