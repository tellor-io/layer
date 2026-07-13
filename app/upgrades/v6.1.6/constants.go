package v6_1_6

import (
	"github.com/tellor-io/layer/app/upgrades"

	store "cosmossdk.io/store/types"

	group "github.com/cosmos/cosmos-sdk/x/group"
)

const (
	UpgradeName = "v6.1.6"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	StoreUpgrades: store.StoreUpgrades{
		Deleted: []string{group.StoreKey},
	},
}
