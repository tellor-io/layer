package v6_1_7

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
Upgrade to v6.1.7 (ADR 1012, reporter power cap):
- New reporter module param max_reporter_power_share caps a single reporter's
  potential stake below a share of total bonded tokens (default 30%).
- Enforcement happens in the TrackStakeChangesDecorator ante handler on
  CreateReporter/SelectReporter/SwitchReporter and on staking messages that
  increase a selector's bonded stake.
- The param deserializes as nil for existing chains, which the ante treats as
  disabled; this handler sets the 0.30 default so the cap activates at upgrade.
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
		if params.MaxReporterPowerShare.IsNil() || params.MaxReporterPowerShare.IsZero() {
			params.MaxReporterPowerShare = reportertypes.DefaultMaxReporterPowerShare
			if err := rk.Params.Set(ctx, params); err != nil {
				return vm, fmt.Errorf("set max_reporter_power_share: %w", err)
			}
			sdkCtx.Logger().Info(
				"set reporter max_reporter_power_share",
				"value", params.MaxReporterPowerShare.String(),
			)
		}

		return vm, nil
	}
}
