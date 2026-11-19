package v6_2_0

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"
)

const (
	UpgradeName = "v6.2.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
