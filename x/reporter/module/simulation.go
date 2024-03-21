package reporter

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/tellor-io/layer/testutil/sample"
	reportersimulation "github.com/tellor-io/layer/x/reporter/simulation"
	"github.com/tellor-io/layer/x/reporter/types"
)

// avoid unused import issue
var (
	_ = reportersimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgCreateReporter = "op_weight_msg_create_reporter"
	// TODO: Determine the simulation weight value
	defaultWeightMsgCreateReporter int = 100

	opWeightMsgDelegateReporter = "op_weight_msg_delegate_reporter"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDelegateReporter int = 100

	opWeightMsgUndelegateReporter = "op_weight_msg_undelegate_reporter"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUndelegateReporter int = 100

	opWeightMsgWithdrawTip = "op_weight_msg_withdraw_tip"
	// TODO: Determine the simulation weight value
	defaultWeightMsgWithdrawTip int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	reporterGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&reporterGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgCreateReporter int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateReporter, &weightMsgCreateReporter, nil,
		func(_ *rand.Rand) {
			weightMsgCreateReporter = defaultWeightMsgCreateReporter
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateReporter,
		reportersimulation.SimulateMsgCreateReporter(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgDelegateReporter int
	simState.AppParams.GetOrGenerate(opWeightMsgDelegateReporter, &weightMsgDelegateReporter, nil,
		func(_ *rand.Rand) {
			weightMsgDelegateReporter = defaultWeightMsgDelegateReporter
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelegateReporter,
		reportersimulation.SimulateMsgDelegateReporter(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUndelegateReporter int
	simState.AppParams.GetOrGenerate(opWeightMsgUndelegateReporter, &weightMsgUndelegateReporter, nil,
		func(_ *rand.Rand) {
			weightMsgUndelegateReporter = defaultWeightMsgUndelegateReporter
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUndelegateReporter,
		reportersimulation.SimulateMsgUndelegateReporter(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgWithdrawTip int
	simState.AppParams.GetOrGenerate(opWeightMsgWithdrawTip, &weightMsgWithdrawTip, nil,
		func(_ *rand.Rand) {
			weightMsgWithdrawTip = defaultWeightMsgWithdrawTip
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgWithdrawTip,
		reportersimulation.SimulateMsgWithdrawTip(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateReporter,
			defaultWeightMsgCreateReporter,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				reportersimulation.SimulateMsgCreateReporter(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgDelegateReporter,
			defaultWeightMsgDelegateReporter,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				reportersimulation.SimulateMsgDelegateReporter(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUndelegateReporter,
			defaultWeightMsgUndelegateReporter,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				reportersimulation.SimulateMsgUndelegateReporter(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgWithdrawTip,
			defaultWeightMsgWithdrawTip,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				reportersimulation.SimulateMsgWithdrawTip(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
