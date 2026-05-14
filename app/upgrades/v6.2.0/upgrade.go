package v6_2_0

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.2.0:
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
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdk.UnwrapSDKContext(ctx).Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
