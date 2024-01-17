package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	mediantypes "github.com/tellor-io/layer/daemons/server/types"
)

var _ = strconv.Itoa(0)

func CmdGetMedianValues() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-median-values",
		Short: "Query getMedianValues",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			fmt.Println(clientCtx.Height)

			queryClient := mediantypes.NewMedianValuesServiceClient(clientCtx)

			params := &mediantypes.GetMedianValuesRequest{}

			res, err := queryClient.GetMedianValues(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
