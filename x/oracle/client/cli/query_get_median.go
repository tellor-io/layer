package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	mediantypes "github.com/tellor-io/layer/daemons/server/types"
	"github.com/tellor-io/layer/utils"
)

var _ = strconv.Itoa(0)

func CmdGetMedianValue() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-median-value",
		Short: "Query getMedianValue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryData := args[0]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := mediantypes.NewMedianValuesServiceClient(clientCtx)

			qData, err := utils.QueryBytesFromString(argQueryData)
			if err != nil {
				return err
			}

			params := &mediantypes.GetMedianValueRequest{QueryData: qData}

			res, err := queryClient.GetMedianValue(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
