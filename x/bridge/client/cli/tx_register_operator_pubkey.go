package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdRegisterOperatorPubkey creates a CLI command for MsgRegisterOperatorPubkey.
func CmdRegisterOperatorPubkey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-operator-pubkey [pubkey]",
		Short: "Register a new operator public key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubkey := args[0]

			msg := types.NewMsgRegisterOperatorPubkey(
				clientCtx.GetFromAddress().String(),
				pubkey,
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
