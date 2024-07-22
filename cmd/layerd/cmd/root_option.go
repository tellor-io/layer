package cmd

import (
	"github.com/spf13/cobra"
	daemonflags "github.com/tellor-io/layer/daemons/flags"
)

// RootCmdOption configures root command option.
type RootCmdOption struct {
	startCmdCustomizer func(*cobra.Command)
}

// newRootCmdOption returns an empty RootCmdOption.
func newRootCmdOption() *RootCmdOption {
	return &RootCmdOption{}
}

// setCustomizeStartCmd accepts a handler to customize the start command and set it in the option.
func (o *RootCmdOption) setCustomizeStartCmd(f func(startCmd *cobra.Command)) {
	o.startCmdCustomizer = f
}

// GetOptionWithCustomStartCmd returns a root command option with custom start commands.
func GetOptionWithCustomStartCmd() *RootCmdOption {
	option := newRootCmdOption()
	f := func(cmd *cobra.Command) {
		// Add daemon flags.
		daemonflags.AddDaemonFlagsToCmd(cmd)
		cmd.Flags().String("keyring-backend", "test", "Select keyring's backend (os|file|kwallet|pass|test)")
		cmd.Flags().String("key-name", "alice", "Select key name")
	}
	option.setCustomizeStartCmd(f)
	return option
}
