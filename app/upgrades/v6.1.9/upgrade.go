package v6_1_9

import (
	"context"
	"fmt"

	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.9 includes (since v6.1.8):
  - Full undelegation sets StakeRecalcFlag. Removals that already happened left
    no flag and the delegation is gone, so every reporter is flagged here. The
    next ReporterStake recalculates once and clears the flag.
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	rk reporterkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		n, err := rk.FlagAllReportersForStakeRecalc(ctx)
		if err != nil {
			return vm, fmt.Errorf("flag reporters for stake recalc: %w", err)
		}
		sdkCtx.Logger().Info("flagged reporters for stake recalculation", "count", n)
		return vm, nil
	}
}
