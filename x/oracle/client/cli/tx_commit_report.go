package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
)

var _ = strconv.Itoa(0)

func CmdCommitReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-report [query-data] [salted-value]",
		Short: "Broadcast message commitReport",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryData := args[0]
			argValue := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			valueDecoded, err := hex.DecodeString(argValue)
			if err != nil {
				return err
			}
			// leaving this here for convenience to input value thru cli
			// then is signed by the keys here
			// data, _, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), valueDecoded, signing.SignMode_SIGN_MODE_DIRECT)
			// if err != nil {
			// 	return err
			// }

			salt, err := oracleutils.Salt(32)
			if err != nil {
				return err
			}

			commit := oracleutils.CalculateCommitment(string(valueDecoded), salt)

			msg := types.NewMsgCommitReport(
				clientCtx.GetFromAddress().String(),
				argQueryData,
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
