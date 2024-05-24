package dispute

import (
	modulev1 "github.com/tellor-io/layer/api/layer/dispute"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "ProposeDispute",
					Use:            "propose-dispute [report] [dispute-category] [fee] [pay-from-bond]",
					Short:          "Execute the ProposeDispute RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "report"}, {ProtoField: "dispute_category"}, {ProtoField: "fee"}, {ProtoField: "pay_from_bond"}},
				},
				{
					RpcMethod:      "AddFeeToDispute",
					Use:            "add-fee-to-dispute [dispute-id] [amount] [pay-from-bond]",
					Short:          "Execute the AddFeeToDispute RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}, {ProtoField: "amount"}, {ProtoField: "pay_from_bond"}},
				},
				{
					RpcMethod:      "Vote",
					Use:            "vote [id] [vote]",
					Short:          "Execute the Vote RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}, {ProtoField: "vote"}},
				},
				{
					RpcMethod:      "TallyVote",
					Use:            "tally-vote [dispute-id]",
					Short:          "Execute the TallyVote RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
				{
					RpcMethod:      "ExecuteDispute",
					Use:            "execute-dispute [dispute-id]",
					Short:          "Execute the ExecuteDispute RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
				{
					RpcMethod: "UpdateTeam",
					Skip:      true, // skipped because team gated
				},
			},
		},
	}
}
