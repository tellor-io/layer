package v_5_0_0

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"
)

const (
	UpgradeName = "v5.0.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
