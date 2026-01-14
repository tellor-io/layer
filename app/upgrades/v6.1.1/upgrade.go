package v6_1_1

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.0 includes:
  - Liveness-weighted Time-Based Rewards (TBR) distribution
  - New oracle parameter: LivenessCycles (controls TBR distribution frequency)
  - TRBBridge queries now receive a share of TBR (as a single slot)
  - New oracle collections for tracking reporter liveness and power shares
  - New reporter collections for distribution queue processing
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
