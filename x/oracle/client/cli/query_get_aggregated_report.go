package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

const (
	FlagBlockNumber = "block-number"
)

func FlagSetBlockNumber() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagBlockNumber, "", "The block number to query the report for.")
	return fs
}

func CmdGetAggregatedReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-aggregated-report [query-id]",
		Short: "Query getAggregatedReport",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqQueryId := args[0]
			reqBlockNumber, err := cmd.Flags().GetInt64(FlagBlockNumber)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetAggregatedReportRequest{

				QueryId:     reqQueryId,
				BlockNumber: reqBlockNumber,
			}

			res, err := queryClient.GetAggregatedReport(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().AddFlagSet(FlagSetBlockNumber())
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
