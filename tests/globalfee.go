package setup

import (
	"github.com/strangelove-ventures/globalfee/x/globalfee"
	globalfeekeeper "github.com/strangelove-ventures/globalfee/x/globalfee/keeper"
	globalfeemodulev1 "github.com/tellor-io/layer/api/layer/globalfee/module"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func init() {
	appmodule.Register(&globalfeemodulev1.Module{},
		appmodule.Provide(FeeProvideModule))
}

type GlobalfeeInputs struct {
	depinject.In

	StoreService *storetypes.KVStoreKey
	Cdc          codec.Codec
	Config       *globalfeemodulev1.Module
}

type GlobalfeeOutputs struct {
	depinject.Out

	GlobalFeekeeper globalfeekeeper.Keeper
	Module          appmodule.AppModule
}

func FeeProvideModule(in GlobalfeeInputs) GlobalfeeOutputs {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := globalfeekeeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		authority.String(),
	)
	m := globalfee.NewAppModule(
		in.Cdc,
		k,
	)
	return GlobalfeeOutputs{GlobalFeekeeper: k, Module: m}
}
