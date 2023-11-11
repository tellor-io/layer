package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

func CmdGetAggregatedReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-aggregated-report [query-id] [block-number]",
		Short: "Query getAggregatedReport",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQueryId := args[0]
			reqBlockNumber := cast.ToInt64(args[1])

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetAggregatedReportRequest{

				QueryId:     reqQueryId,
				BlockNumber: reqBlockNumber,
			}

			res, err := queryClient.GetAggregatedReport(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
