package cli

import (
	"encoding/hex"
	"fmt"
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

			queryId, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}

			timestampStr := args[1]
			timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timestamp %s", timestampStr)
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetSnapshotsByReportRequest{QueryId: queryId, Timestamp: uint64(timestampInt)}

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
