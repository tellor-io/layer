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
					RpcMethod: "GetAllDataSpecs",
					Use:       "all-data-specs",
					Short:     "Shows all registered data specs",
				},
				{
					RpcMethod:      "DecodeQuerydata",
					Use:            "decode-querydata [query_data]",
					Short:          "Decode the query data into human readable format",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}},
				},

				{
					RpcMethod:      "GenerateQuerydata",
					Use:            "generate-querydata [querytype] [parameters]",
					Short:          "Encode query data hex given query type and parameters",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "querytype"}, {ProtoField: "parameters"}},
				},

				{
					RpcMethod:      "DecodeValue",
					Use:            "decode-value [query-type] [value]",
					Short:          "Decode the value given query type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "queryType"}, {ProtoField: "value"}},
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
