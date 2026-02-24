package v6_1_2_patch

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"
)

const (
	UpgradeName = "v6.1.2-patch"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
