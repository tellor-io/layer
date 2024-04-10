package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetAttestationsBySnapshot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-attestations-by-snapshot [snapshot]",
		Short: "Query get-attestations-by-snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			snapshot := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetAttestationsBySnapshotRequest{Snapshot: snapshot}

			res, err := queryClient.GetAttestationsBySnapshot(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
