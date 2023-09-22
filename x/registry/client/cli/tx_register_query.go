package cli

import (
	"strconv"

	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/registry/types"
)

var _ = strconv.Itoa(0)

func CmdRegisterQuery() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-query [query-type] [data-types] [data-fields]",
		Short: "Broadcast message registerQuery",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryType := args[0]
			argDataTypes := strings.Split(args[1], listSeparator)
			argDataFields := strings.Split(args[2], listSeparator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterQuery(
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
