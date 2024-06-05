package bridge

import (
	modulev1 "github.com/tellor-io/layer/api/layer/bridge"

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
					RpcMethod:      "GetEvmValidators",
					Use:            "get-evm-validators",
					Short:          "Query all EVM validators",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetValidatorCheckpoint",
					Use:            "get-validator-checkpoint",
					Short:          "Query validator checkpoint",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetValidatorCheckpointParams",
					Use:            "get-validator-checkpoint-params [timestamp]",
					Short:          "Query validator checkpoint params given a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetValidatorTimestampByIndex",
					Use:            "get-validator-timestamp-by-index [index]",
					Short:          "Query validator timestamp by index",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
				{
					RpcMethod:      "GetValsetSigs",
					Use:            "get-valset-sigs [timestamp]",
					Short:          "Query valset signatures by timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetEvmAddressByValidatorAddress",
					Use:            "get-evm-address-by-validator-address [validator_address]",
					Short:          "Query EVM address by validator address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}},
				},
				{
					RpcMethod:      "GetValsetByTimestamp",
					Use:            "get-valset-by-timestamp [timestamp]",
					Short:          "Query valset by timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetCurrentAggregateReport",
					Use:            "get-current-aggregate-report [query_id]",
					Short:          "Query current aggregate report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}},
				},
				{
					RpcMethod:      "GetDataBefore",
					Use:            "get-data-before [query-id] [timestamp]",
					Short:          "Query data before a timestamp",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetSnapshotsByReport",
					Use:            "get-snapshots-by-report [query-id] [timestamp]",
					Short:          "Query snapshots by report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "GetAttestationDataBySnapshot",
					Use:            "get-attestation-data-by-snapshot [snapshot]",
					Short:          "Query snapshots by report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "snapshot"}},
				},
				{
					RpcMethod:      "GetAttestationsBySnapshot",
					Use:            "get-attestation-by-snapshot [snapshot]",
					Short:          "Query snapshots by report",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "snapshot"}},
				},
				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "RequestAttestations",
					Use:            "request-attestations [creator] [query_id] [timestamp]",
					Short:          "Execute the RequestAttestations RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}, {ProtoField: "query_id"}, {ProtoField: "timestamp"}},
				},
				{
					RpcMethod:      "WithdrawTokens",
					Use:            "withdraw-tokens [creator] [recipient] [amount]",
					Short:          "Execute the WithdrawTokens RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}, {ProtoField: "recipient"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod:      "ClaimDeposit",
					Use:            "claim-deposit [creator] [deposit-id] [index]",
					Short:          "Execute the ClaimDeposit RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}, {ProtoField: "deposit_id"}, {ProtoField: "index"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
