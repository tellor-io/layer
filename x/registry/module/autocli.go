package registry

import (
	modulev1 "github.com/tellor-io/layer/api/layer/registry"

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
					RpcMethod:      "GetDataSpec",
					Use:            "data-spec [query-type]",
					Short:          "Shows the data spec for the given query type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_type"}},
				},
				{
					RpcMethod:      "DecodeQuerydata",
					Use:            "decode-querydata [query-data]",
					Short:          "Decode the query data into human readable format",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}},
				},

				{
					RpcMethod:      "GenerateQuerydata",
					Use:            "generate-querydata [query-type] [parameters]",
					Short:          "Encode query data hex given query type and parameters",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_type"}, {ProtoField: "parameters"}},
				},

				{
					RpcMethod:      "DecodeValue",
					Use:            "decode-value [query-type] [value]",
					Short:          "Decode the value given query type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_type"}, {ProtoField: "value"}},
				},

				// this line is used by ignite scaffolding # autocli/query
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
				{
					RpcMethod: "RemoveDataSpecs",
					Skip:      true, // skipped because authority gated
				},
			},
		},
	}
}
