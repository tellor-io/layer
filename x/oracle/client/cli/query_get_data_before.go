package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

func CmdGetDataBefore() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-data-before [query-id] [timestamp]",
		Short: "Query get data before",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQueryId := args[0]
			if err != nil {
				return err
			}
			reqTimestamp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetDataBeforeRequest{
				QueryId:   reqQueryId,
				Timestamp: reqTimestamp,
			}

			res, err := queryClient.GetDataBefore(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
