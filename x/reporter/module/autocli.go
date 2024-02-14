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
					Short:          "Send a createReporter tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "amount"}, {ProtoField: "token_origins"}},
				},
				{
					RpcMethod:      "DelegateReporter",
					Use:            "delegate-reporter [reporter] [amount] [token-origin]",
					Short:          "Send a delegateReporter tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter"}, {ProtoField: "amount"}, {ProtoField: "token_origins"}},
				},
				{
					RpcMethod:      "UndelegateReporter",
					Use:            "undelegate-reporter [amount]",
					Short:          "Send a undelegateReporter tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "token_origins"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
