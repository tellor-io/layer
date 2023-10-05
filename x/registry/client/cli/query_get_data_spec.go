package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/registry/types"
)

var _ = strconv.Itoa(0)

func CmdGetDataSpec() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-data-spec [query-type]",
		Short: "Query getDataSpec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQueryType := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetDataSpecRequest{

				QueryType: reqQueryType,
			}

			res, err := queryClient.GetDataSpec(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
