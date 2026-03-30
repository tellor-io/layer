package fork

import (
	"context"
	"fmt"

	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
	Upgrade to fork will include:
		- reading in module state data from json files that were created and exported from the previous version

	Fork backfill (dispute, reporter, oracle) runs via MigrateFork on each keeper migrator.
	We do not call mm.RunMigrations here so consensus versions stay aligned with mainnet.
*/

func CreateUpgradeHandler(
	disputeKeeper disputekeeper.Keeper,
	reporterKeeper reporterkeeper.Keeper,
	oracleKeeper oraclekeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		if err := disputekeeper.NewMigrator(disputeKeeper).MigrateFork(sdkCtx); err != nil {
			return nil, fmt.Errorf("dispute fork migration: %w", err)
		}
		if err := reporterkeeper.NewMigrator(reporterKeeper).MigrateFork(sdkCtx); err != nil {
			return nil, fmt.Errorf("reporter fork migration: %w", err)
		}
		if err := oraclekeeper.NewMigrator(oracleKeeper).MigrateFork(sdkCtx); err != nil {
			return nil, fmt.Errorf("oracle fork migration: %w", err)
		}

		return vm, nil
	}
}
