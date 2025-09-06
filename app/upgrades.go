package app

import (
	"fmt"

	"github.com/tellor-io/layer/app/upgrades"
	v_5_1_1 "github.com/tellor-io/layer/app/upgrades/v5.1.1"
	v_5_1_2 "github.com/tellor-io/layer/app/upgrades/v5.1.2"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

var (
	// `Upgrades` defines the upgrade handlers and store loaders for the application.
	// New upgrades should be added to this slice after they are implemented.
	Upgrades = []*upgrades.Upgrade{
		&v_5_1_1.Upgrade,
	}
	Forks = []upgrades.Fork{}
)

// setupUpgradeHandlers registers the upgrade handlers to perform custom upgrade
// logic and state migrations for software upgrades.
func (app *App) setupUpgradeHandlers() {
	if app.UpgradeKeeper.HasHandler(v_5_1_2.UpgradeName) {
		panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", v_5_1_2.UpgradeName))
	}
	app.UpgradeKeeper.SetUpgradeHandler(
		v_5_1_2.UpgradeName,
		v_5_1_2.CreateUpgradeHandler(
			app.ModuleManager(),
			app.configurator,
			app.AccountKeeper,
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
