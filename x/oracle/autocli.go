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
					Short:          "Query all reports by query id.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetReportsbyReporter",
					Use:            "get-reportsby-reporter [reporter]",
					Short:          "Query reports by reporter with or without pagination",
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
					RpcMethod:      "GetTippedQueries",
					Use:            "get-tipped-queries",
					Short:          "Query to get all available tipped queries",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "TippedQueriesForDaemon",
					Use:            "tipped-queries-for-daemon",
					Short:          "Query to get all available tipped queries (for daemon's eyes only)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetCurrentQueryByQueryId",
					Use:            "get-current-query-by-query-id [query_id]",
					Short:          "Query current query by query id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetQueryDataLimit",
					Use:            "get-query-data-limit",
					Short:          "Query query data limit",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "ReportedIdsByReporter",
					Use:            "reported_ids_by_reporter [reporter_address]",
					Short:          "Query reported ids by reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "GetCycleList",
					Use:            "get-cycle-list",
					Short:          "Query cycle list",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetReportersNoStakeReports",
					Use:            "get-reporters-no-stake-reports [reporter]",
					Short:          "Query no stake reports by reporter with or without pagination",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter"}},
				},
				{
					RpcMethod:      "GetNoStakeReportsByQueryId",
					Use:            "get-no-stake-reports-by-query-id [query_id]",
					Short:          "Query no stake reports by query id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetTipTotal",
					Use:            "get-tip-total",
					Short:          "Query total tips",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
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
					Use:            "submit-value [qdata] [value]",
					Short:          "Execute the SubmitValue RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}, {ProtoField: "value"}},
				},
				{
					RpcMethod:      "Tip",
					Use:            "tip [query_data] [amount]",
					Short:          "Execute the Tip RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod: "UpdateCyclelist",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod: "UpdateQueryDataLimit",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "NoStakeReport",
					Use:            "no-stake-report [query_data] [value]",
					Short:          "Execute the NoStakeReport RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_data"}, {ProtoField: "value"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
