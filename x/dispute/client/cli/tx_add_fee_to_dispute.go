package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/dispute/types"
)

var _ = strconv.Itoa(0)

func CmdAddFeeToDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-fee-to-dispute [dispute-id] [amount] [pay-from-bond] [validator-address]",
		Short: "Broadcast message addFeeToDispute",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fs := cmd.Flags()
			argDisputeId, err := cast.ToUint64E(args[0])
			if err != nil {
				return err
			}
			argAmount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			payFromBond, _ := fs.GetBool(FlagPayFromBond)

			validatorAddress, _ := fs.GetString(FlagAddressValidator)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddFeeToDispute(
				clientCtx.GetFromAddress().String(),
				argDisputeId,
				argAmount,
				payFromBond,
				validatorAddress,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
