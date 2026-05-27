package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

// layerEnvClientFlags are flags commonly set via LAYER_* environment variables.
var layerEnvClientFlags = []string{
	flags.FlagHome,
	flags.FlagNode,
	flags.FlagChainID,
	flags.FlagKeyringBackend,
	flags.FlagKeyringDir,
}

// applyLayerEnvToFlags copies LAYER_* environment variables (registered on the global
// viper instance by server/cmd.Execute) onto command flags when the user did not set
// the flag on the command line. Without this, ReadPersistentCommandFlags keeps a
// pre-populated client.Context home and ignores env vars.
func applyLayerEnvToFlags(cmd *cobra.Command) error {
	for _, name := range layerEnvClientFlags {
		f, fs := lookupInheritedFlag(cmd, name)
		if f == nil || f.Changed {
			continue
		}
		if !viper.IsSet(name) {
			continue
		}
		val := viper.GetString(name)
		if val == "" {
			continue
		}
		if err := fs.Set(name, val); err != nil {
			return fmt.Errorf("set flag %q from %s env: %w", name, EnvPrefix, err)
		}
	}
	return nil
}

func lookupInheritedFlag(cmd *cobra.Command, name string) (*pflag.Flag, *pflag.FlagSet) {
	for c := cmd; c != nil; c = c.Parent() {
		for _, fs := range []*pflag.FlagSet{c.Flags(), c.PersistentFlags()} {
			if f := fs.Lookup(name); f != nil {
				return f, fs
			}
		}
	}
	return nil, nil
}
