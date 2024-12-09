package upgrades

import (
	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

type Upgrade struct {
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator) upgradetypes.UpgradeHandler
	// Store Upgrades, should be used for any new modules introduced, deleted, or store names renamed
	StoreUpgrades store.StoreUpgrades
}

type Fork struct {
	// Upgrade version name, for the upgrade handler, e.g. `v2.0.0-audit`
	UpgradeName string

	// Height the upgrade occurs at
	UpgradeHeight func(*module.Manager, module.Configurator) upgradetypes.UpgradeHandler

	// Upgrade info for this fork
	UpgradeInfo string
}
