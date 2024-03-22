package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

var _ = strconv.Itoa(0)

func CmdGetEvmAddressByValidatorAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-evm-address-by-validator-address [validator-address]",
		Short: "Query get-evm-address-by-validator-address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			validatorAddress := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetEvmAddressByValidatorAddressRequest{ValidatorAddress: validatorAddress}

			res, err := queryClient.GetEvmAddressByValidatorAddress(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
