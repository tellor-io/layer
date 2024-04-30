package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdClaimDeposit creates a CLI command for MsgClaimDeposit.
func CmdClaimDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-deposit [deposit-id] [report-index]",
		Short: "Claim a deposit from ethereum to layer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			depositId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			reportIndex, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			msg := types.NewMsgClaimDepositRequest(
				clientCtx.GetFromAddress().String(),
				depositId,
				reportIndex,
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
