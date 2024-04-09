package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetDataBefore() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-data-before [queryId] [timestamp]",
		Short: "Query get-data-before",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]
			timestamp := args[1]
			timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			qIdBz, err := utils.QueryBytesFromString(queryId)
			if err != nil {
				return err
			}

			params := &types.QueryGetDataBeforeRequest{QueryId: qIdBz, Timestamp: timestampInt}

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
