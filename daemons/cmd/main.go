package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/daemons"
	"github.com/tellor-io/layer/daemons/configs"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
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
		logLevelstr := viper.GetString(flags.FlagLogLevel)
		configs.WriteDefaultPricefeedExchangeToml(homePath)
		configs.WriteDefaultMarketParamsToml(homePath)
		customquery.WriteDefaultConfigToml(homePath, "config", "custom_query_config.toml")
		loglevel, err := zerolog.ParseLevel(logLevelstr)
		if err != nil {
			fmt.Printf("Error parsing log level: %v\n", err)
			os.Exit(1)
		}
		logger := log.NewLogger(os.Stderr, log.LevelOption(loglevel))

		// Check if test mode is enabled
		if testMode {
			if err := runTestMode(homePath, logger); err != nil {
				fmt.Printf("Test mode failed: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		// Normal daemon mode - validate required flags
		chainId := viper.GetString(flags.FlagChainID)
		grpcAddr := viper.GetString(flags.FlagGRPC)
		from := viper.GetString(flags.FlagFrom)
		node := viper.GetString(flags.FlagNode)

		if chainId == "" {
			fmt.Printf("Error: --chain-id is required in reporter mode\n")
			os.Exit(1)
		}
		if grpcAddr == "" {
			fmt.Printf("Error: --grpc is required in reporter mode\n")
			os.Exit(1)
		}
		if from == "" {
			fmt.Printf("Error: --from is required in reporter mode\n")
			os.Exit(1)
		}
		if node == "" {
			fmt.Printf("Error: --node is required in reporter mode\n")
			os.Exit(1)
		}

		// Set up signal handling for graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		// Pass prometheusPort and signal context to NewApp
		appInstance := daemons.NewApp(ctx, logger, chainId, grpcAddr, homePath, prometheusPort)

		// Wait for signal
		<-ctx.Done()
		logger.Info("Received shutdown signal, shutting down gracefully...")

		// Gracefully shutdown
		appInstance.Shutdown()
	},
}

var prometheusPort int
var testMode bool

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
	rootCmd.Flags().String(flags.FlagLogLevel, zerolog.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>')")
	rootCmd.Flags().String(flags.FlagBroadcastMode, flags.BroadcastSync, "Transaction broadcasting mode (sync|async)")
	rootCmd.Flags().String(flags.FlagNode, "", "<host>:<port> to CometBFT RPC interface for layer")
	rootCmd.Flags().IntVar(&prometheusPort, "prometheus-port", 26661, "Port to serve Prometheus metrics on (default 26661). Applicable only if telemetry is enabled in app.toml.")

	// Price Guard Flags
	rootCmd.Flags().Bool("price-guard-enabled", false, "Enable price guard to prevent reporting prices that differ from last reported price by a given threshold")
	rootCmd.Flags().Float64("price-guard-threshold", 0, "Price change threshold (0.5 = 50%, 0.01 = 1% (up to 15 decimals)) - submissions exceeding this will be blocked")
	rootCmd.Flags().Duration("price-guard-max-age", 0, "Maximum age of stored price before treating as expired (e.g. 1m, 1h)")
	rootCmd.Flags().Bool("price-guard-update-on-blocked", false, "Update last known price even if submission is blocked (default false)")

	// Test mode flag
	rootCmd.Flags().BoolVar(&testMode, "test", false, "Test mode: verify price feed configurations and calculate medians without starting daemon")
	// Automatic Unbonding flags
	rootCmd.Flags().Uint32("auto-unbonding-frequency", 0, "Enable automatic unbonding every N days (0 = disabled, 1 - 21 days = valid")
	rootCmd.Flags().Uint32("auto-unbonding-amount", 0, "Amount of tokens in loya to unbond each unbonding transaction (0 = disabled)")
	rootCmd.Flags().String("auto-unbonding-max-stake-percentage", "0.0", "Maximum percentage of stake to unbond each unbonding transaction (0 = disabled, 1.0 = 100%). If unbonding amount exceeds this percentage, we will skip the unbonding transaction until it exceeds this percentage again.")

	// Marking required flags
	if err := rootCmd.MarkFlagRequired(flags.FlagHome); err != nil {
		panic(err)
	}
	// Note: --from, --grpc, --chain-id, and --node are only required in normal mode, not test mode
	// We'll validate them in the Run function instead

	// Try to load .env from current directory, or parent directory if not found
	if err := godotenv.Load(); err != nil {
		// Try parent directory (for when running from daemons/ subdirectory)
		if err := godotenv.Load("../.env"); err != nil {
			// .env file is optional, so we don't panic if it's not found
			// This allows the daemon to run without .env if environment variables are set another way
		}
	}

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		panic(err)
	}
}
