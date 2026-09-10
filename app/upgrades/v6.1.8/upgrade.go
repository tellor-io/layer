package v6_1_8

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.8 includes (since v6.1.7):
  - LastConsensusTimestampByQueryId map: written in CreateNewReportSnapshots
    after the valset/power-threshold update, including aggregates past
    SnapshotLimit that are not snapshotted. GetLastConsensusTimestamp no
    longer walks prior snapshots. Missing prior snapshots no longer fail
    CreateSnapshot / EndBlock.
  - The last-consensus map is backfilled by bridge 5→6 from the latest
    consensus aggregate per queryId (current PowerThreshold, flagged included);
    live writes are in CreateNewReportSnapshots; prefix 24; StoreUpgrades empty.
  - Oracle cycle-list: persist the live query data in CurrentCycleListQuery (prefix 47)
    so GetCurrentQueryInCycleList no longer indexes GetCyclelist()[Peek()]. Oracle
    consensus version 5→6 backfills the item from the sequencer with a wrap on OOB.
    StoreUpgrades stays empty (new prefix in the existing oracle store).
No other custom state migration is required beyond RunMigrations.
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
