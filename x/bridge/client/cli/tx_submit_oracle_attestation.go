package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CmdSubmitOracleAttestation creates a CLI command for MsgSubmitOracleAttestation.
func CmdSubmitOracleAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-oracle-attestation [queryId] [timestamp] [signature]",
		Short: "Submit a new oracle attestation",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryId := args[0]
			timestamp := args[1]
			signature := args[2]

			msg := types.NewMsgSubmitOracleAttestation(
				clientCtx.GetFromAddress().String(),
				queryId,
				timestamp,
				signature,
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
