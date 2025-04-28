package v_4_0_3

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"
)

const (
	UpgradeName = "v4.0.3"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
