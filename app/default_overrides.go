package app

import (
	"encoding/json"

	icq "github.com/cosmos/ibc-apps/modules/async-icq/v8"
	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
	"github.com/strangelove-ventures/globalfee/x/globalfee"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// bankModule defines a custom wrapper around the x/bank module's AppModuleBasic
// implementation to provide custom default genesis state.
type bankModule struct {
	bank.AppModule
}

// DefaultGenesis returns custom x/bank module genesis state.
func (bankModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	metadata := banktypes.Metadata{
		Description: "The native token of the Tellor Layer.",
		Base:        BondDenom,
		Name:        DisplayDenom,
		Display:     DisplayDenom,
		Symbol:      DisplayDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    BondDenom, // ie Wei
				Exponent: 0,
			},
			{
				Denom:    DisplayDenom, // ie Ether
				Exponent: 6,
			},
		},
	}

	genState := banktypes.DefaultGenesisState()
	genState.DenomMetadata = append(genState.DenomMetadata, metadata)

	return cdc.MustMarshalJSON(genState)
}

// stakingModule wraps the x/staking module in order to overwrite specific
// ModuleManager APIs.
type stakingModule struct {
	staking.AppModule
}

// DefaultGenesis returns custom x/staking module genesis state.
func (stakingModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	params := stakingtypes.DefaultParams()
	params.BondDenom = BondDenom
	return cdc.MustMarshalJSON(&stakingtypes.GenesisState{
		Params: params,
	})
}

type distrModule struct {
	distribution.AppModule
}

// DefaultGenesis returns custom x/distribution module genesis state.
func (distrModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := distrtypes.DefaultGenesisState()
	genState.Params.CommunityTax = math.LegacyZeroDec() // 0% community tax on gas fees, inflation is minted to timeBasedRewards for reporters

	return cdc.MustMarshalJSON(genState)
}

type govModule struct {
	gov.AppModuleBasic
}

// DefaultGenesis returns custom x/gov module genesis state.
func (govModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := govtypes.DefaultGenesisState()
	genState.Params.MinDeposit = sdk.Coins{sdk.NewInt64Coin(BondDenom, 10000000)}
	genState.Params.ExpeditedMinDeposit = sdk.Coins{sdk.NewInt64Coin(BondDenom, 50000000)}

	return cdc.MustMarshalJSON(genState)
}

type globalFeeModule struct {
	*globalfee.AppModule
}

// DefaultGenesis returns custom x/globalfee module genesis state.
func (globalFeeModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := globalfeetypes.DefaultGenesisState()
	genState.Params.MinimumGasPrices = sdk.NewDecCoins(sdk.NewDecCoinFromDec(BondDenom, math.LegacyNewDecWithPrec(25, 4)))

	return cdc.MustMarshalJSON(genState)
}

type icqcustomModule struct {
	icq.AppModule
}

// DefaultGenesis returns custom x/globalfee module genesis state.
func (icqcustomModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := icqtypes.DefaultGenesis()
	genState.Params.AllowQueries = []string{"/layer.oracle.Query/GetCurrentAggregateReport"}

	return cdc.MustMarshalJSON(genState)
}
