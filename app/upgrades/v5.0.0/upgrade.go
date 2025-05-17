package v_5_0_0

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
	Upgrade to v5.0.0 will include:
		- Update statechange.yml by @themandalore in #691
		- chore: cleanup evm and deployment by @tkernell in #692
		- Update README.md by @themandalore in #689
		- fix release workflow by @akremstudy in #694
		- chore: rename blobstream contract by @tkernell in #695
		- [Backport release/v5.x] Edit release workflow by @github-actions in #698
		- [Backport release/v5.x] feat: allow selectors to become reporters, reporters to become selectors by @github-actions in #701
		- [Backport release/v5.x] Test/social fork by @github-actions in #702
		- [Backport release/v5.x] feat: no stake reporting by @github-actions in #706
		- [Backport release/v5.x] chore: stop cosmos if keyring not accessible by @github-actions in #707
	Migrations:
		- None
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
