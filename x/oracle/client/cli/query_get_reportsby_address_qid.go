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

func CmdGetReportsbyAddressQid() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-reports-by-address-qid [address] [qid]",
		Short: "Query CmdGetReportsbyAddressQid",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAddress := args[0]
			reqQid := args[1]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			qIdBz, err := utils.QueryBytesFromString(reqQid)
			if err != nil {
				return err
			}

			params := &types.QueryGetReportsbyReporterQidRequest{

				Reporter: reqAddress,
				QueryId:  qIdBz,
			}

			res, err := queryClient.GetReportsbyReporterQid(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
