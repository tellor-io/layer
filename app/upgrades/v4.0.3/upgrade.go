package v_4_0_3

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
	Upgrade to v4.0.3 will include:
		- split the daemon into a separate binary
		- added stuff to vote extensions to deal with a problem with validators submitting vote extensions without any votes
		- updated cli to return a hex string instead of a base64 string in most cases
		- bridge deposits will no longer need finality but a number of blocks to pass on ethereum before the deposit can be reported for

	Migrations:
		- Dispute:
			- MigrateStore: update disputes with ID 2 and 3 to be closed since they were resolved
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
