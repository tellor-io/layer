package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	mediantypes "github.com/tellor-io/layer/daemons/server/types"
	"github.com/tellor-io/layer/lib/prices"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
)

var _ = strconv.Itoa(0)

func CmdCommitSubmitReport() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-submit-report [query-data]",
		Short: "Broadcast message report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argQueryData := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			queryClient := mediantypes.NewMedianValuesServiceClient(clientCtx)

			params := &mediantypes.GetMedianValueRequest{QueryData: argQueryData}

			medianValue, err := queryClient.GetMedianValue(cmd.Context(), params)
			if err != nil {
				return err
			}

			if medianValue == nil {
				return fmt.Errorf("no median values found for query data: %s", argQueryData)
			}

			fmt.Println("Spot Price:", medianValue.MedianValues.Price)
			var hexValue string
			hexValue, err = prices.EncodePrice(float64(medianValue.MedianValues.Price), medianValue.MedianValues.Exponent)
			if err != nil {
				return err
			}
			fmt.Println("Hex Value:", hexValue)
			valueDecoded, err := hex.DecodeString(hexValue)
			if err != nil {
				return err
			}
			// data, _, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), valueDecoded, signing.SignMode_SIGN_MODE_DIRECT)
			// if err != nil {
			// 	return err
			// }
			// get salted value
			salt, err := utils.Salt(32)
			if err != nil {
				return err
			}

			commit := utils.CalculateCommitment(string(valueDecoded), salt)

			msg := types.NewMsgCommitReport(
				clientCtx.GetFromAddress().String(),
				argQueryData,
				commit,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			if err = tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg); err != nil {
				panic(err)
			}
			time.Sleep(time.Second)
			msgSubmit := types.NewMsgSubmitValue(
				clientCtx.GetFromAddress().String(),
				argQueryData,
				hexValue,
				salt,
			)
			if err := msgSubmit.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgSubmit)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
