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
		cmd.Flags().String("key-name", "", "Select key name")
		// Remote-signer mode for vote-extension signing (no local key on this node).
		cmd.Flags().String("remote-signer-addr", "", "remote signer gRPC address host:port; if set, vote-extensions are signed via the remote signer instead of a local keyring")
		cmd.Flags().String("remote-signer-ca-cert", "", "mTLS CA cert path for the remote signer (required unless --remote-signer-insecure)")
		cmd.Flags().String("remote-signer-client-cert", "", "mTLS client cert path for the remote signer (required unless --remote-signer-insecure)")
		cmd.Flags().String("remote-signer-client-key", "", "mTLS client key path for the remote signer (required unless --remote-signer-insecure)")
		cmd.Flags().String("remote-signer-server-name", "bridge-signer", "expected CN in the remote signer's TLS cert")
		cmd.Flags().Bool("remote-signer-insecure", false, "explicitly allow a plaintext, unauthenticated connection to the remote signer (local/test only)")
	}
	option.setCustomizeStartCmd(f)
	return option
}
