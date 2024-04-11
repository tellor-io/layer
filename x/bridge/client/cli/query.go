package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/tellor-io/layer/x/bridge/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group bridge queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdGetEvmValidators())
	cmd.AddCommand(CmdGetValidatorCheckpoint())
	cmd.AddCommand(CmdGetValidatorCheckpointParams())
	cmd.AddCommand(CmdGetValidatorTimestampByIndex())
	cmd.AddCommand(CmdGetValsetSigs())
	cmd.AddCommand(CmdGetEvmAddressByValidatorAddress())
	cmd.AddCommand(CmdGetValsetByTimestamp())
	cmd.AddCommand(CmdGetCurrentAggregateReport())
	cmd.AddCommand(CmdGetDataBefore())
	cmd.AddCommand(CmdGetSnapshotsByReport())
	cmd.AddCommand(CmdGetAttestationDataBySnapshot())
	cmd.AddCommand(CmdGetAttestationsBySnapshot())
	// this line is used by starport scaffolding # 1

	return cmd
}
