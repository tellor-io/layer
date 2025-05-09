package app

import (
	"fmt"

	"github.com/tellor-io/layer/app/upgrades"
	v_4_0_3 "github.com/tellor-io/layer/app/upgrades/v4.0.3"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

var (
	// `Upgrades` defines the upgrade handlers and store loaders for the application.
	// New upgrades should be added to this slice after they are implemented.
	Upgrades = []*upgrades.Upgrade{
		&v_4_0_3.Upgrade,
	}
	Forks = []upgrades.Fork{}
)

// setupUpgradeHandlers registers the upgrade handlers to perform custom upgrade
// logic and state migrations for software upgrades.
func (app *App) setupUpgradeHandlers() {
	if app.UpgradeKeeper.HasHandler(v_4_0_3.UpgradeName) {
		panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", v_4_0_3.UpgradeName))
	}
	app.UpgradeKeeper.SetUpgradeHandler(
		v_4_0_3.UpgradeName,
		v_4_0_3.CreateUpgradeHandler(
			app.ModuleManager(),
			app.configurator,
		),
	)
}

// setUpgradeStoreLoaders sets custom store loaders to customize the rootMultiStore
// initialization for software upgrades.
func (app *App) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades))
		}
	}
}
