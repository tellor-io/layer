package v6_1_6

import (
	"context"
	"fmt"

	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.6:
- Deferred reporter switches stored in OutgoingPendingSwitches / IncomingPendingSwitchIdx
  with ReporterPendingSwitchHeads for O(1) checks.
- Finalization runs at the start of ReporterStake (e.g. when a reporter submits),
  not in BeginBlock.
- Pending switch targets live only in keeper collections (not on Selection).
  Max pending switches per reporter is a module param (default 10).

No custom state migration is required beyond RunMigrations: new collections and
proto fields deserialize to empty / zero for existing chains.
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

		params, err := rk.Params.Get(ctx)
		if err != nil {
			return vm, fmt.Errorf("reporter params: %w", err)
		}
		if params.MaxPendingSwitchesPerReporter == 0 {
			params.MaxPendingSwitchesPerReporter = reportertypes.DefaultMaxPendingSwitchesPerReporter
			if err := rk.Params.Set(ctx, params); err != nil {
				return vm, fmt.Errorf("set max_pending_switches_per_reporter: %w", err)
			}
			sdkCtx.Logger().Info(
				"set reporter max_pending_switches_per_reporter",
				"value", params.MaxPendingSwitchesPerReporter,
			)
		}

		return vm, nil
	}
}
