package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetCurrentAggregateReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-current-aggregate-report [queryId]",
		Short: "Query get-oracle-attestations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetCurrentAggregateReportRequest{QueryId: queryId}

			res, err := queryClient.GetCurrentAggregateReport(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
