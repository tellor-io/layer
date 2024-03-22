package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetOracleAttestations() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-oracle-attestations [queryId] [timestamp]",
		Short: "Query get-oracle-attestations",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]

			timestamp, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetOracleAttestationsRequest{QueryId: queryId, Timestamp: timestamp}

			res, err := queryClient.GetOracleAttestations(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
