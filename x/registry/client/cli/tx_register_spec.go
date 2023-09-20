package cli

import (
	"strconv"

	"encoding/json"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"layer/x/registry/types"
)

var _ = strconv.Itoa(0)

func CmdRegisterSpec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-spec [query-type] [spec]",
		Short: "Broadcast message registerSpec",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryType := args[0]
			argSpec := new(types.DataSpec)
			err = json.Unmarshal([]byte(args[1]), argSpec)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterSpec(
				clientCtx.GetFromAddress().String(),
				argQueryType,
				argSpec,
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
