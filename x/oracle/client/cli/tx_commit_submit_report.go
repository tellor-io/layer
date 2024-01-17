package cli

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/spf13/cobra"
	mediantypes "github.com/tellor-io/layer/daemons/server/types"
	"github.com/tellor-io/layer/x/oracle/types"
)

var _ = strconv.Itoa(0)

var (
	queryDataIdMap = map[string]uint32{
		"00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000": 0,
		"00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000": 1,
	}
)

func EncodeValue(number float64) (string, error) {
	strNumber := fmt.Sprintf("%.18f", number)

	parts := strings.Split(strNumber, ".")
	if len(parts[1]) > 18 {
		parts[1] = parts[1][:18]
	}
	truncatedStr := parts[0] + parts[1]

	bigIntNumber := new(big.Int)
	_, ok := bigIntNumber.SetString(truncatedStr, 10)
	if !ok {
		return "", fmt.Errorf("error converting string to big int")
	}

	uint256ABIType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return "", fmt.Errorf("error creating uint256 abi type, %v", err)
	}

	arguments := abi.Arguments{{Type: uint256ABIType}}
	encodedBytes, err := arguments.Pack(bigIntNumber)
	if err != nil {
		return "", fmt.Errorf("error packing arguments, %v", err)
	}

	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString, nil
}
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

			params := &mediantypes.GetMedianValuesRequest{}

			medianValue, err := queryClient.GetMedianValues(cmd.Context(), params)
			if err != nil {
				return err
			}
			if len(medianValue.MedianValues) == 0 {
				return fmt.Errorf("no median values found")
			}
			var hexValue string
			for _, value := range medianValue.MedianValues {
				if queryDataIdMap[argQueryData] == value.MarketId {
					fmt.Println("Spot Price:", value.Price)
					hexValue, err = EncodeValue(float64(value.Price))
					if err != nil {
						return err
					}
					break
				}
				return fmt.Errorf("query data not found in median values")
			}
			valueDecoded, err := hex.DecodeString(hexValue)
			if err != nil {
				return err
			}
			data, _, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), valueDecoded)
			if err != nil {
				return err
			}

			msg := types.NewMsgCommitReport(
				clientCtx.GetFromAddress().String(),
				argQueryData,
				hex.EncodeToString(data),
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
