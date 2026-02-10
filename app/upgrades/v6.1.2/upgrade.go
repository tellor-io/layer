package v6_1_2

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.2 includes:
  - Reporter stake caching: skip full recalculation when delegation state hasn't changed
  - New reporter collections: LastValSetUpdateHeight, StakeRecalcFlag, RecalcAtTime
  - Staking hooks now flag reporters for recalculation on validator set changes and delegation modifications
  - Microreport pruning: oracle EndBlocker removes reports older than 30 days (batched, max 100/block)
  - Simplified reporter PruneOldReports using oracle block-height lookup
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
