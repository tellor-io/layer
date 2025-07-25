package reporter

import (
	"fmt"

	"github.com/spf13/cobra"
	modulev1 "github.com/tellor-io/layer/api/layer/reporter"
	"github.com/tellor-io/layer/x/reporter/types"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
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
					RpcMethod:      "Reporters",
					Use:            "reporters",
					Short:          "Query staked reporters",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "SelectorReporter",
					Use:            "selector-reporter [selector-address]",
					Short:          "Query reporter of a selector",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "selector_address"}},
				},
				{
					RpcMethod:      "AllowedAmount",
					Use:            "allowed-amount [reporter-address]",
					Short:          "Query current allowed amount to stake or unstake",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "NumOfSelectorsByReporter",
					Use:            "num-of-selectors-by-reporter [reporter-address]",
					Short:          "Query how many selectors a reporter has",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "SpaceAvailableByReporter",
					Use:            "space-available-by-reporter [reporter-address]",
					Short:          "Query how much space a reporter has",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "AvailableTips",
					Use:            "available-tips [selector-address]",
					Short:          "Query how much tips a selector has",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "selector_address"}},
				},
				{
					RpcMethod:      "SelectionsTo",
					Use:            "selections-to [reporter-address]",
					Short:          "Query the selectors for a reporter",
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
					Use:            "create-reporter [commission-rate] [min-tokens-required] [moniker]",
					Short:          "Execute the CreateReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "commission_rate"}, {ProtoField: "min_tokens_required"}, {ProtoField: "moniker"}},
				},
				{
					RpcMethod:      "SelectReporter",
					Use:            "select-reporter [reporter-address]",
					Short:          "Execute the SelectReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "SwitchReporter",
					Use:            "switch-reporter [reporter-address]",
					Short:          "Execute the SwitchReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "reporter_address"}},
				},
				{
					RpcMethod:      "RemoveSelector",
					Use:            "remove-selector [selector-address]",
					Short:          "Execute the RemoveSelector RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "selector_address"}},
				},
				{
					RpcMethod:      "UnjailReporter",
					Use:            "unjail-reporter",
					Short:          "Execute the UnjailReporter RPC method",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod: "WithdrawTip",
					Use:       "withdraw-tip [selector-address] [validator-address]",
					Short:     "Send a WithdrawTip tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "selector_address"},
						{ProtoField: "validator_address"},
					},
				},
				{
					RpcMethod:      "EditReporter",
					Use:            "edit-reporter [commission-rate] [min-tokens-required] [moniker]",
					Short:          "edit commission rate, moniker, and MinTokensRequired for your reporter",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "commission_rate"}, {ProtoField: "min_tokens_required"}, {ProtoField: "moniker"}},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}

func (AppModule) GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "reporter",
		Short:              "Transactions command for the reporter module",
		RunE:               client.ValidateCmd,
		DisableFlagParsing: true,
	}
	cmd.AddCommand(GetTxCreateReporterCmd())
	return cmd
}

func GetTxCreateReporterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-reporter [commission-rate] [min-tokens-required] [moniker]",
		Short: "Execute the CreateReporter RPC method",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			minTokensRequired, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid min-tokens-required: %s", args[1])
			}

			msg := types.MsgCreateReporter{
				ReporterAddress:   clientCtx.FromAddress.String(),
				CommissionRate:    math.LegacyMustNewDecFromStr(args[0]),
				MinTokensRequired: minTokensRequired,
				Moniker:           args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}
	cmd.Flags().Bool("genesis", false, "if true will print the json init message for genesis")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
