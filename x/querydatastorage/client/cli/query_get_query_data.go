package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"layer/x/querydatastorage/types"
)

var _ = strconv.Itoa(0)

func CmdGetQueryData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-query-data [query-id]",
		Short: "Query getQueryData",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQueryId := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetQueryDataRequest{

				QueryId: reqQueryId,
			}

			res, err := queryClient.GetQueryData(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
