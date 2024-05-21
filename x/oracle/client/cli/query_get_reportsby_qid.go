package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

func CmdGetReportsbyQid() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-reportsby-qid [q-id]",
		Short: "Query getReportsbyQid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQId := utils.Remove0xPrefix(args[0])

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetReportsbyQidRequest{
				QueryId: reqQId,
			}

			res, err := queryClient.GetReportsbyQid(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
