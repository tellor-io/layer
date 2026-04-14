package cmd

import (
	"github.com/spf13/cobra"
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
		cmd.Flags().String("keyring-backend", "test", "Select keyring's backend (os|file|kwallet|pass|test)")
		cmd.Flags().String("key-name", "alice", "Select key name")
		cmd.Flags().Bool("remote-signer-enabled", false, "Use bridge-signer sidecar instead of local keyring")
		cmd.Flags().String("remote-signer-addr", "", "gRPC address of the bridge-signer sidecar")
		cmd.Flags().String("remote-signer-ca-cert", "", "Path to CA cert for verifying the sidecar")
		cmd.Flags().String("remote-signer-client-cert", "", "Path to validator's client TLS cert")
		cmd.Flags().String("remote-signer-client-key", "", "Path to validator's client TLS key")
		cmd.Flags().String("remote-signer-server-name", "bridge-signer", "Expected CN in sidecar's TLS cert")
	}
	option.setCustomizeStartCmd(f)
	return option
}
