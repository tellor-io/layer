package oracle

import (
	"math/rand"

	"github.com/tellor-io/layer/testutil/sample"
	oraclesimulation "github.com/tellor-io/layer/x/oracle/simulation"
	"github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = oraclesimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgSubmitValue = "op_weight_msg_submit_value"
	// TODO: Determine the simulation weight value
	defaultWeightMsgSubmitValue int = 100

	opWeightMsgTip = "op_weight_msg_tip"
	// TODO: Determine the simulation weight value
	defaultWeightMsgTip int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	oracleGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&oracleGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgSubmitValue int
	simState.AppParams.GetOrGenerate(opWeightMsgSubmitValue, &weightMsgSubmitValue, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitValue = defaultWeightMsgSubmitValue
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubmitValue,
		oraclesimulation.SimulateMsgSubmitValue(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgTip int
	simState.AppParams.GetOrGenerate(opWeightMsgTip, &weightMsgTip, nil,
		func(_ *rand.Rand) {
			weightMsgTip = defaultWeightMsgTip
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgTip,
		oraclesimulation.SimulateMsgTip(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgSubmitValue,
			defaultWeightMsgSubmitValue,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				oraclesimulation.SimulateMsgSubmitValue(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgTip,
			defaultWeightMsgTip,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				oraclesimulation.SimulateMsgTip(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
