package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
)

var _ = strconv.Itoa(0)

func CmdCommitReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-report [query-id] [signature]",
		Short: "Broadcast message commitReport",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryId := args[0]
			argValue := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			// valueDecoded, err := hex.DecodeString(argValue)
			// if err != nil {
			// 	return err
			// }
			// leaving this here for convenience to input value thru cli
			// then is signed by the keys here
			// data, _, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), valueDecoded, signing.SignMode_SIGN_MODE_DIRECT)
			// if err != nil {
			// 	return err
			// }

			salt, err := utils.Salt(32)
			if err != nil {
				return err
			}

			commit := utils.CalculateCommitment(argValue, salt)

			msg := types.NewMsgCommitReport(
				clientCtx.GetFromAddress().String(),
				argQueryId,
				commit,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			cmd.Println("Copy your salt for the reveal stage:", salt) //

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
