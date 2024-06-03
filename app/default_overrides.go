package app

import (
	"encoding/json"

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

// DefaultGenesis returns custom x/distribution module genesis state.
func (govModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := govtypes.DefaultGenesisState()
	genState.Params.MinDeposit = sdk.Coins{sdk.NewInt64Coin(BondDenom, 10000000)}
	genState.Params.ExpeditedMinDeposit = sdk.Coins{sdk.NewInt64Coin(BondDenom, 50000000)}

	return cdc.MustMarshalJSON(genState)
}
