package app

import (
	"fmt"

	"github.com/tellor-io/layer/app/upgrades"
<<<<<<< HEAD
	v_6_1_4 "github.com/tellor-io/layer/app/upgrades/v6.1.4"
=======
	v_6_1_3 "github.com/tellor-io/layer/app/upgrades/v6.1.3"
>>>>>>> 2d1e37f962adb56bca35f67313a988c6d6611fd5

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

var (
	// `Upgrades` defines the upgrade handlers and store loaders for the application.
	// New upgrades should be added to this slice after they are implemented.
	Upgrades = []*upgrades.Upgrade{
<<<<<<< HEAD
		&v_6_1_4.Upgrade,
=======
		&v_6_1_3.Upgrade,
>>>>>>> 2d1e37f962adb56bca35f67313a988c6d6611fd5
	}
	Forks = []upgrades.Fork{}
)

// setupUpgradeHandlers registers the upgrade handlers to perform custom upgrade
// logic and state migrations for software upgrades.
func (app *App) setupUpgradeHandlers() {
<<<<<<< HEAD
	if app.UpgradeKeeper.HasHandler(v_6_1_4.UpgradeName) {
		panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", v_6_1_4.UpgradeName))
	}
	app.UpgradeKeeper.SetUpgradeHandler(
		v_6_1_4.UpgradeName,
		v_6_1_4.CreateUpgradeHandler(
=======
	if app.UpgradeKeeper.HasHandler(v_6_1_3.UpgradeName) {
		panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", v_6_1_3.UpgradeName))
	}
	app.UpgradeKeeper.SetUpgradeHandler(
		v_6_1_3.UpgradeName,
		v_6_1_3.CreateUpgradeHandler(
>>>>>>> 2d1e37f962adb56bca35f67313a988c6d6611fd5
			app.ModuleManager(),
			app.configurator,
			app.OracleKeeper,
			app.BridgeKeeper,
			app.RegistryKeeper,
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
