package v_3_0_4

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
	Upgrade to v3.0.4 will include:
		- an update to the reporter daemon that fixes an issue we were occasionally seeing when reporting bridge deposits
		- add a field (LastConsensusTimestamp) to bridge type AttestationSnapshotData to track the last
		- added more telemetry statements to track chain performance and oracle usage/stats

	Migrations:
		- Bridge module: Iterates through AttestationSnapshotData field and adds the new field set to 0
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
