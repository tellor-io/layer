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
				{
					RpcMethod:      "GetSnapshotLimit",
					Use:            "get-snapshot-limit",
					Short:          "Query snapshot limit",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetDepositClaimed",
					Use:            "get-deposit-claimed [deposit_id]",
					Short:          "Query deposit claimed",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "deposit_id"}},
				},
				{
					RpcMethod:      "GetLastWithdrawalId",
					Use:            "get-last-withdrawal-id",
					Short:          "Query last withdrawal id",
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
					RpcMethod:      "ClaimDeposits",
					Use:            "claim-deposits [deposit-ids] [timestamps]",
					Short:          "Execute the ClaimDeposits RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "deposit_ids"}, {ProtoField: "timestamps"}},
				},
				{
					RpcMethod:      "UpdateSnapshotLimit",
					Use:            "update-snapshot-limit [limit]",
					Short:          "Execute the UpdateSnapshotLimit RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "limit"}},
				},
				{
					RpcMethod:      "SubmitAttestationEvidence",
					Use:            "submit-attestation-evidence [creator] [query_id] [value] [timestamp] [aggregate_power] [previous_timestamp] [next_timestamp] [valset_checkpoint] [attestation_timestamp] [last_consensus_timestamp] [signature] [operator_address]",
					Short:          "Execute the SubmitAttestationEvidence RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "creator"}, {ProtoField: "query_id"}, {ProtoField: "value"}, {ProtoField: "timestamp"}, {ProtoField: "aggregate_power"}, {ProtoField: "previous_timestamp"}, {ProtoField: "next_timestamp"}, {ProtoField: "valset_checkpoint"}, {ProtoField: "attestation_timestamp"}, {ProtoField: "last_consensus_timestamp"}, {ProtoField: "signature"}, {ProtoField: "operator_address"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}

// type MsgSubmitAttestationEvidence struct {
//     Creator                string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
//     QueryId                string `protobuf:"bytes,2,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
//     Value                  string `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
//     Timestamp              uint64 `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
//     AggregatePower         uint64 `protobuf:"varint,5,opt,name=aggregate_power,json=aggregatePower,proto3" json:"aggregate_power,omitempty"`
//     PreviousTimestamp      uint64 `protobuf:"varint,6,opt,name=previous_timestamp,json=previousTimestamp,proto3" json:"previous_timestamp,omitempty"`
//     NextTimestamp          uint64 `protobuf:"varint,7,opt,name=next_timestamp,json=nextTimestamp,proto3" json:"next_timestamp,omitempty"`
//     ValsetCheckpoint       string `protobuf:"bytes,8,opt,name=valset_checkpoint,json=valsetCheckpoint,proto3" json:"valset_checkpoint,omitempty"`
//     AttestationTimestamp   uint64 `protobuf:"varint,9,opt,name=attestation_timestamp,json=attestationTimestamp,proto3" json:"attestation_timestamp,omitempty"`
//     LastConsensusTimestamp uint64 `protobuf:"varint,10,opt,name=last_consensus_timestamp,json=lastConsensusTimestamp,proto3" json:"last_consensus_timestamp,omitempty"`
//     Signature              string `protobuf:"bytes,11,opt,name=signature,proto3" json:"signature,omitempty"`
//     OperatorAddress        string `protobuf:"bytes,12,opt,name=operator_address,json=operatorAddress,proto3" json:"operator_address,omitempty"`
// }
