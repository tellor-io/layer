package cli

import (
    "strconv"
	
	"github.com/spf13/cobra"
    "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

func CmdGetCycleList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-cycle-list",
		Short: "Query getCycleList",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetCycleListRequest{
				
            }

            

			res, err := queryClient.GetCycleList(cmd.Context(), params)
            if err != nil {
                return err
            }

            return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

    return cmd
}
