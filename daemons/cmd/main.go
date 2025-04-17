package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/daemons"
	"github.com/tellor-io/layer/daemons/configs"
	daemonflags "github.com/tellor-io/layer/daemons/flags"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "reporterd",
	Short: "Run reporter daemon",
	Long:  "reporterd is a daemon that runs the reporter that interacts with the layer chain.",
	Run: func(cmd *cobra.Command, args []string) {
		homePath := viper.GetString(flags.FlagHome)
		chainId := viper.GetString(flags.FlagChainID)
		grpcAddr := viper.GetString(flags.FlagGRPC)
		configs.WriteDefaultPricefeedExchangeToml(homePath)
		configs.WriteDefaultMarketParamsToml(homePath)
		logger := log.NewLogger(os.Stderr, log.LevelOption(zerolog.InfoLevel))
		daemons.NewApp(logger, chainId, grpcAddr, homePath)
	},
}

func main() {
	daemonflags.AddDaemonFlagsToCmd(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "Node home directory")
	rootCmd.Flags().String(flags.FlagFrom, "", "Name of the key to use")
	rootCmd.Flags().String(flags.FlagGRPC, "0.0.0.0:9090", "Address to listen on")
	rootCmd.Flags().String(flags.FlagChainID, "layer", "Chain ID")
	rootCmd.Flags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test|memory)")
	rootCmd.PersistentFlags().String(flags.FlagLogLevel, zerolog.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>')")
	rootCmd.Flags().String(flags.FlagBroadcastMode, flags.BroadcastSync, "Transaction broadcasting mode (sync|async)")
	rootCmd.Flags().String(flags.FlagNode, "", "<host>:<port> to CometBFT RPC interface for layer")

	// Marking required flags
	if err := rootCmd.MarkFlagRequired(flags.FlagHome); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkFlagRequired(flags.FlagFrom); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkFlagRequired(flags.FlagGRPC); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkFlagRequired(flags.FlagChainID); err != nil {
		panic(err)
	}
	if err := rootCmd.MarkFlagRequired(flags.FlagNode); err != nil {
		panic(err)
	}

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		panic(err)
	}
}
