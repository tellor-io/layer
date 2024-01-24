package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	mediantypes "github.com/tellor-io/layer/daemons/server/types"
)

var _ = strconv.Itoa(0)

func CmdGetAllMedianValues() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-all-median-values",
		Short: "Query getAllMedianValues",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := mediantypes.NewMedianValuesServiceClient(clientCtx)

			params := &mediantypes.GetAllMedianValuesRequest{}

			res, err := queryClient.GetAllMedianValues(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
