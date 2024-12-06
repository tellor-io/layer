package v_2_0_0_alpha2

import (
	store "cosmossdk.io/store/types"

	"github.com/tellor-io/layer/app/upgrades"
)

const (
	UpgradeName = "v2.0.0-audit"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:   UpgradeName,
	StoreUpgrades: store.StoreUpgrades{},
}
