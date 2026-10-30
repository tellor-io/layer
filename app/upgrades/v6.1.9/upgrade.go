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
  - Oracle time-based rewards: out-of-turn tips on cycle-list queries no longer
    over-credit a query's slot. The tip handler marks the new window with
    QueryMeta.out_of_turn (new proto field 9) instead of demoting the query and
    counting the extra opportunity at tip time; both now happen when the window
    aggregates, so the accounting lands in the liveness period the window is
    actually in. DemoteQueryToNonStandard no longer overwrites the per-query
    opportunity count, rotation clears the mark when it adopts a tipped window
    as the turn, and DistributeLivenessRewards counts slots from the queries
    that had a window in the period rather than the live cycle-list length.
  - Oracle: a TRBBridge deposit aggregate counts toward the bridge rewards slot
    only when its power meets the bridge validator-set power threshold.
  - Reporter: the BeforeDelegationRemoved staking hook flags a stake recalc, so
    a full undelegation invalidates the reporter's cached stake snapshot the
    same way a partial one does.
  - Reporter: JailReporter flags a stake recalc for the jailed reporter, so its
    next report takes the recount path where the jailed check lives.

State notes:
  - The handler sets LastValSetUpdateHeight to the upgrade height so every
    reporter recounts stake on its next report. Any reporter holding a stale
    cached snapshot at upgrade time (fully unbonded or jailed before the fixes
    above existed) is corrected on first contact instead of waiting for an
    unrelated validator-set event.
  - out_of_turn is wire-compatible; every stored QueryMeta decodes it as false,
    which is correct for windows created before the upgrade because the old
    code already applied their demotion and opportunity at tip time. No
    backfill is done on purpose: marking them would count them a second time.
  - No new store prefixes, StoreUpgrades empty, no module consensus version
    bump. No custom state migration is required beyond RunMigrations.
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	rk reporterkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		if err := rk.LastValSetUpdateHeight.Set(ctx, uint64(sdkCtx.BlockHeight())); err != nil {
			return vm, err
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
