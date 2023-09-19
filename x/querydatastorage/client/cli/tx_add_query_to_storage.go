package cli

import (
	"strconv"
	"strings"

	"layer/x/querydatastorage/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdAddQueryToStorage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-query-to-storage [query-type] [data-types] [data-fields]",
		Short: "Broadcast message addQueryToStorage",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryType := args[0]
			argDataTypes := strings.Split(args[1], ",")
			argDataFields := strings.Split(args[2], ",")

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddQueryToStorage(
				clientCtx.GetFromAddress().String(),
				argQueryType,
				argDataTypes,
				argDataFields,
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
