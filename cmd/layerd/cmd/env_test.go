package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

func TestApplyLayerEnvToFlags_Home(t *testing.T) {
	t.Helper()
	viper.Reset()
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	t.Cleanup(viper.Reset)

	aliceHome := t.TempDir()
	t.Setenv("LAYER_HOME", aliceHome)

	root := &cobra.Command{Use: "layerd"}
	root.PersistentFlags().String(flags.FlagHome, "/default/home", "node home")
	leaf := &cobra.Command{Use: "keys"}
	root.AddCommand(leaf)

	require.NoError(t, applyLayerEnvToFlags(leaf))

	home, err := root.PersistentFlags().GetString(flags.FlagHome)
	require.NoError(t, err)
	require.Equal(t, aliceHome, home)
}

func TestApplyLayerEnvToFlags_CLIOverridesEnv(t *testing.T) {
	t.Helper()
	viper.Reset()
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	t.Cleanup(viper.Reset)

	t.Setenv("LAYER_HOME", "/from/env")

	root := &cobra.Command{Use: "layerd"}
	root.PersistentFlags().String(flags.FlagHome, "/default/home", "node home")
	leaf := &cobra.Command{Use: "keys"}
	root.AddCommand(leaf)
	require.NoError(t, root.PersistentFlags().Set(flags.FlagHome, "/from/cli"))

	require.NoError(t, applyLayerEnvToFlags(leaf))

	home, err := root.PersistentFlags().GetString(flags.FlagHome)
	require.NoError(t, err)
	require.Equal(t, "/from/cli", home)
}

func TestApplyLayerEnvToFlags_UnsetEnv(t *testing.T) {
	t.Helper()
	viper.Reset()
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	t.Cleanup(viper.Reset)

	require.NoError(t, os.Unsetenv("LAYER_HOME"))

	root := &cobra.Command{Use: "layerd"}
	root.PersistentFlags().String(flags.FlagHome, "/default/home", "node home")
	leaf := &cobra.Command{Use: "keys"}
	root.AddCommand(leaf)

	require.NoError(t, applyLayerEnvToFlags(leaf))

	home, err := root.PersistentFlags().GetString(flags.FlagHome)
	require.NoError(t, err)
	require.Equal(t, "/default/home", home)
}
