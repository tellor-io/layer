package fork

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"
)

const (
	UpgradeName = "fork"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
