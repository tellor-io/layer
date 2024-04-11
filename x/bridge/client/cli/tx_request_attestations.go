package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdRequestAttestations creates a CLI command for MsgRequestAttestations.
func CmdRequestAttestations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-attestations [queryId] [timestamp]",
		Short: "Request attestations for a given queryId and timestamp",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]
			timestamp := args[1]

			msg := types.NewMsgRequestAttestations(
				clientCtx.GetFromAddress().String(),
				queryId,
				timestamp,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
