package dispute

import (
	"math/rand"

	"github.com/tellor-io/layer/testutil/sample"
	disputesimulation "github.com/tellor-io/layer/x/dispute/simulation"
	"github.com/tellor-io/layer/x/dispute/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = disputesimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgProposeDispute          = "op_weight_msg_propose_dispute"
	defaultWeightMsgProposeDispute int = 100

	opWeightMsgAddFeeToDispute          = "op_weight_msg_add_fee_to_dispute"
	defaultWeightMsgAddFeeToDispute int = 100

	opWeightMsgVote          = "op_weight_msg_vote"
	defaultWeightMsgVote int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	disputeGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&disputeGenesis)
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

	var weightMsgProposeDispute int
	simState.AppParams.GetOrGenerate(opWeightMsgProposeDispute, &weightMsgProposeDispute, nil,
		func(_ *rand.Rand) {
			weightMsgProposeDispute = defaultWeightMsgProposeDispute
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgProposeDispute,
		disputesimulation.SimulateMsgProposeDispute(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgAddFeeToDispute int
	simState.AppParams.GetOrGenerate(opWeightMsgAddFeeToDispute, &weightMsgAddFeeToDispute, nil,
		func(_ *rand.Rand) {
			weightMsgAddFeeToDispute = defaultWeightMsgAddFeeToDispute
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgAddFeeToDispute,
		disputesimulation.SimulateMsgAddFeeToDispute(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgVote int
	simState.AppParams.GetOrGenerate(opWeightMsgVote, &weightMsgVote, nil,
		func(_ *rand.Rand) {
			weightMsgVote = defaultWeightMsgVote
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgVote,
		disputesimulation.SimulateMsgVote(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgProposeDispute,
			defaultWeightMsgProposeDispute,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				disputesimulation.SimulateMsgProposeDispute(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgAddFeeToDispute,
			defaultWeightMsgAddFeeToDispute,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				disputesimulation.SimulateMsgAddFeeToDispute(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgVote,
			defaultWeightMsgVote,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				disputesimulation.SimulateMsgVote(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
