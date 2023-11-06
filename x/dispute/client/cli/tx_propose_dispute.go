package cli

import (
	"strconv"

	flag "github.com/spf13/pflag"

	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/dispute/types"
)

var _ = strconv.Itoa(0)

const (
	FlagAddressValidator = "validator-address"
	FlagPayFromBond      = "pay-from-bond"
)

func FlagSetValidatorAddress() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagAddressValidator, "", "The validator address where the delegation is to pool stake from.")
	return fs
}

func FlagSetPayFromBond() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagPayFromBond, "", "Boolean wether to pay from bond or not.")
	return fs
}
func CmdProposeDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-dispute [report] [dispute-category] [fee]",
		Short: "Broadcast message proposeDispute",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			fs := cmd.Flags()
			argReport := new(types.MicroReport)
			err = json.Unmarshal([]byte(args[0]), argReport)
			if err != nil {
				return err
			}
			argDisputeCategory := new(types.DisputeCategory)
			err = json.Unmarshal([]byte(args[1]), argDisputeCategory)
			if err != nil {
				return err
			}
			argFee, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}

			payFromBond, _ := fs.GetBool(FlagPayFromBond)

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgProposeDispute(
				clientCtx.GetFromAddress().String(),
				argReport,
				*argDisputeCategory,
				argFee,
				payFromBond,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().AddFlagSet(FlagSetPayFromBond())
	cmd.Flags().AddFlagSet(FlagSetValidatorAddress())
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
