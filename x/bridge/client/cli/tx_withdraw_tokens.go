package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdWithdrawTokens creates a CLI command for MsgWithdrawTokens.
func CmdWithdrawTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-tokens [recipient] [amount]",
		Short: "Withdraw tokens from layer to ethereum",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			recipient := args[0]
			amount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			msg := types.NewMsgWithdrawTokens(
				clientCtx.GetFromAddress().String(),
				recipient,
				amount,
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
