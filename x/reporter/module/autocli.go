package reporter

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/tellor-io/layer/api/layer/reporter"
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
					RpcMethod:      "Reporter",
					Use:            "reporter [reporter-addr]",
					Short:          "Query staked reporter by address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "Reporters",
					Use:            "reporters",
					Short:          "Query staked reporters",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "DelegatorReporter",
					Use:            "delegator-reporter [delegator-addr]",
					Short:          "Query reporter of a delegator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}},
				},
				{
					RpcMethod:      "ReporterStake",
					Use:            "reporter-stake [reporter-addr]",
					Short:          "Query total tokens of a reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "DelegationRewards",
					Use:            "delegation-rewards [delegator-addr] [reporter-addr]",
					Short:          "Query delegator rewards from a particular reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}, {ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "ReporterOutstandingRewards",
					Use:            "outstanding-rewards [reporter]",
					Short:          "Query outstanding rewards for a reporter and all their delegations",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "ReporterCommission",
					Use:            "commission [reporter]",
					Short:          "Query distribution reporter commission",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "CreateReporter",
					Use:            "create-reporter [amount] [token-origins]",
					Short:          "Execute the CreateReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "amount"}, {ProtoField: "token_origins"}},
				},
				{
					RpcMethod:      "DelegateReporter",
					Use:            "delegate-reporter [reporter] [amount] [token-origin]",
					Short:          "Execute the DelegateReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter"}, {ProtoField: "amount"}, {ProtoField: "token_origins"}},
				},
				{
					RpcMethod:      "UndelegateReporter",
					Use:            "undelegate-reporter [amount]",
					Short:          "Execute the UndelegateReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "token_origins"}},
				},
				{
					RpcMethod:      "UnjailReporter",
					Use:            "unjail-reporter [reporter-addr]",
					Short:          "Execute the UnjailReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "WithdrawTip",
					Use:            "withdraw-tip [delegator-address] [validator-address]",
					Short:          "Send a WithdrawTip tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "delegator_address"}, {ProtoField: "validator_address"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
