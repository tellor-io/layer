package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdSubmitBridgeValsetSignature creates a CLI command for MsgSubmitBridgeValsetSignature.
func CmdSubmitBridgeValsetSignature() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-bridge-valset-signature [timestamp] [signature]",
		Short: "Submit a new bridge valset signature",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			timestamp := args[0]
			signature := args[1]

			msg := types.NewMsgSubmitBridgeValsetSignature(
				clientCtx.GetFromAddress().String(),
				timestamp,
				signature,
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
