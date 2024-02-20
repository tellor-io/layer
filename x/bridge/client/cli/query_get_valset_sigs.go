package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetValsetSigs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-valset-sigs [timestamp]",
		Short: "Query get-valset-sigs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			timestamp, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetValsetSigsRequest{Timestamp: timestamp}

			res, err := queryClient.GetValsetSigs(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
