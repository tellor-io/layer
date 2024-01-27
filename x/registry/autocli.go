package registry

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "github.com/tellor-io/layer/api/layer/registry"
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
					RpcMethod:      "GetDataSpec",
					Use:            "get-data-spec [query-type]",
					Short:          "Shows the data spec for the given query type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_type"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateDataSpec",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "RegisterSpec",
					Use:            "register-spec [query-type] [spec]",
					Short:          "Broadcast message RegisterSpec",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_type"}, {ProtoField: "spec"}},
				},
			},
		},
	}
}
