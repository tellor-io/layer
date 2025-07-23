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
				{
					RpcMethod:      "TeamVote",
					Use:            "team-vote [dispute-id]",
					Short:          "Shows the team vote for a dispute",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
				{
					RpcMethod:      "Tally",
					Use:            "tally [dispute-id]",
					Short:          "Shows the tally for a dispute",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
				{
					RpcMethod:      "Disputes",
					Use:            "disputes",
					Short:          "Shows all disputes",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "OpenDisputes",
					Use:            "open-disputes",
					Short:          "Shows all open disputes",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "TeamAddress",
					Use:            "team-address",
					Short:          "Shows the team address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "VoteResult",
					Use:            "vote-result [dispute-id]",
					Short:          "Shows the vote result for a dispute",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "ProposeDispute",
					Use:            "propose-dispute [disputed-reporter] [report-meta-id] [report-query-id] [dispute-category] [fee] [pay-from-bond]",
					Short:          "Execute the ProposeDispute RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "disputed_reporter"}, {ProtoField: "report_meta_id"}, {ProtoField: "report_query_id"}, {ProtoField: "dispute_category"}, {ProtoField: "fee"}, {ProtoField: "pay_from_bond"}},
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
					RpcMethod:      "WithdrawFeeRefund",
					Use:            "withdraw-fee-refund [payer-address] [id]",
					Short:          "Execute the WithdrawFeeRefund RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "payer_address"}, {ProtoField: "id"}},
				},
				{
					RpcMethod:      "UpdateTeam",
					Use:            "update-team [new-team-address]",
					Short:          "Execute the UpdateTeam RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "new_team_address"}},
				},
				{
					RpcMethod:      "ClaimReward",
					Use:            "claim-reward [dispute_id]",
					Short:          "Execute the ClaimReward RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}},
				},
				{
					RpcMethod:      "AddEvidence",
					Use:            "add-evidence [dispute-id] [reports]",
					Short:          "Execute the AddEvidence RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "dispute_id"}, {ProtoField: "reports"}},
				},
			},
		},
	}
}
