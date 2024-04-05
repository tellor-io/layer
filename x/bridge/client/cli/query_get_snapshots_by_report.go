package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetSnapshotsByReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-snapshots-by-report [queryId] [timestamp]",
		Short: "Query get-snapshots-by-report",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]
			timestamp := args[1]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetSnapshotsByReportRequest{QueryId: queryId, Timestamp: timestamp}

			res, err := queryClient.GetSnapshotsByReport(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
