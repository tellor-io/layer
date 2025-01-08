package oracle

import (
	modulev1 "github.com/tellor-io/layer/api/layer/oracle"

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
					RpcMethod:      "GetReportsbyQid",
					Use:            "get-reportsby-qid [query-id]",
					Short:          "Query all reports by query id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetReportsbyReporter",
					Use:            "get-reportsby-reporter [reporter]",
					Short:          "Query reports by reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter"}},
				},
				{
					RpcMethod:      "GetReportsbyReporterQid",
					Use:            "get-reportsby-reporter-qid [reporter] [query-id]",
					Short:          "Query total tokens of a reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter"}, {ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetReportsByAggregate",
					Use:            "get-reports-by-aggregate [query_id] [timestamp]",
					Short:          "Query reports by aggregate",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetCurrentTip",
					Use:            "get-current-tip [query_data]",
					Short:          "Query current tip for a query",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}},
				},
				{
					RpcMethod:      "GetUserTipTotal",
					Use:            "get-user-tip-total [tipper]",
					Short:          "Query tippers total tipped amount",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "tipper"}},
				},
				{
					RpcMethod:      "GetDataBefore",
					Use:            "get-data-before [query_id] [timestamp]",
					Short:          "Query data before a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetDataAfter",
					Use:            "get-data-after [query_id] [timestamp]",
					Short:          "Query data after a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "RetrieveData",
					Use:            "retrieve-data [query_id] [timestamp]",
					Short:          "get data for a query at a specific timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetTimeBasedRewards",
					Use:            "get-time-based-rewards",
					Short:          "Query time based rewards in system",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "CurrentCyclelistQuery",
					Use:            "current-cyclelist-query",
					Short:          "Query current query in cycle list",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "NextCyclelistQuery",
					Use:            "next-cyclelist-query",
					Short:          "Query next query in cycle list",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetCurrentAggregateReport",
					Use:            "get-current-aggregate-report [query_id]",
					Short:          "Query current aggregate report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetAggregateBeforeByReporter",
					Use:            "get-aggregate-before-by-reporter [query_id] [timestamp] [reporter]",
					Short:          "Query aggregate before a timestamp by reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}, {ProtoField: "reporter"}},
				},
				{
					RpcMethod:      "TippedQueries",
					Use:            "tipped-queries",
					Short:          "Query to get all available tipped queries",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetCurrentQueryByQueryId",
					Use:            "get-current-query-by-query-id [query_id]",
					Short:          "Query current query by query id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "ReportedIdsByReporter",
					Use:            "reported_ids_by_reporter [reporter_address]",
					Short:          "Query reported ids by reporter",
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
					RpcMethod:      "SubmitValue",
					Use:            "submit-value [creator] [qdata] [value] [salt]",
					Short:          "Execute the SubmitValue RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}, {ProtoField: "query_data"}, {ProtoField: "value"}},
				},
				{
					RpcMethod:      "Tip",
					Use:            "tip [tipper] [query_data] [amount]",
					Short:          "Execute the Tip RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "tipper"}, {ProtoField: "query_data"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod: "UpdateCyclelist",
					Skip:      true, // skipped because authority gated
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
