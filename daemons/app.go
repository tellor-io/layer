package daemons

import (
	"context"
	"os"
	"runtime/debug"
	"time"

	"github.com/spf13/cast"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/daemons/configs"
	"github.com/tellor-io/layer/daemons/constants"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
	daemonflags "github.com/tellor-io/layer/daemons/flags"
	metricsclient "github.com/tellor-io/layer/daemons/metrics/client"
	pricefeedclient "github.com/tellor-io/layer/daemons/pricefeed/client"
	reporterclient "github.com/tellor-io/layer/daemons/reporter/client"
	daemonserver "github.com/tellor-io/layer/daemons/server"
	daemonservertypes "github.com/tellor-io/layer/daemons/server/types"
	pricefeedtypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	tokenbridgetypes "github.com/tellor-io/layer/daemons/server/types/token_bridge"
	tokenbridgetipstypes "github.com/tellor-io/layer/daemons/server/types/token_bridge_tips"
	tokenbridgeclient "github.com/tellor-io/layer/daemons/token_bridge_feed/client"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	"google.golang.org/grpc"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

type App struct {
	Server              *daemonserver.Server
	PriceFeedClient     *pricefeedclient.Client
	ReporterClient      *reporterclient.Client
	DaemonHealthMonitor *daemonservertypes.HealthMonitor
	TokenBridgeClient   *tokenbridgeclient.Client
}

// NewAppWithPrometheusPort allows specifying the prometheus port.
func NewApp(
	logger log.Logger,
	chainId,
	grpcAddress,
	homePath string,
	prometheusPort int,
) {
	RegisterTelemetryIfEnabled(logger, homePath, prometheusPort)
	tempDir := func() string {
		dir, err := os.MkdirTemp("", "tellorapp")
		if err != nil {
			dir = app.DefaultNodeHome
		}
		defer os.RemoveAll(dir)

		return dir
	}
	appOpts := simtestutil.NewAppOptionsWithFlagHome(tempDir())
	daemonFlags := daemonflags.GetDaemonFlagValuesFromOptions(appOpts)
	queries, err := customquery.BuildQueryEndpoints(homePath, "config", "custom_query_config.toml")
	if err != nil {
		panic(err)
	}

	indexPriceCache := pricefeedtypes.NewMarketToExchangePrices(constants.MaxPriceAge)

	tokenDepositsCache := tokenbridgetypes.NewDepositReports()
	tokenBridgeTipsCache := tokenbridgetipstypes.NewDepositTips()
	// Create server that will ingest gRPC messages from daemon clients.
	// Note that gRPC clients will block on new gRPC connection until the gRPC server is ready to
	// accept new connections.
	server := daemonserver.NewServer(
		logger,
		grpc.NewServer(),
		&daemontypes.FileHandlerImpl{},
		daemonFlags.Shared.SocketAddress,
	)

	server.WithPriceFeedMarketToExchangePrices(indexPriceCache)
	daemonHealthMonitor := daemonservertypes.NewHealthMonitor(
		daemonservertypes.DaemonStartupGracePeriod,
		daemonservertypes.HealthCheckPollFrequency,
		logger,
		daemonFlags.Shared.PanicOnDaemonFailureEnabled,
	)
	server.WithTokenDepositsCache(tokenDepositsCache)
	// Create a closure for starting pricefeed daemon and daemon server. Daemon services are delayed until after the gRPC
	// service is started because daemons depend on the gRPC service being available. If a node is initialized
	// with a genesis time in the future, then the gRPC service will not be available until the genesis time, the
	// daemons will not be able to connect to the cosmos gRPC query service and finish initialization, and the daemon
	// monitoring service will panic.

	// set flag `--price-daemon-max-unhealthy-seconds=0` to disable
	maxDaemonUnhealthyDuration := time.Duration(daemonFlags.Shared.MaxDaemonUnhealthySeconds) * time.Second
	// Start server for handling gRPC messages from daemons.
	go server.Start()

	exchangeQueryConfig := configs.ReadExchangeQueryConfigFile(homePath)
	marketParamsConfig := configs.ReadMarketParamsConfigFile(homePath)
	// Start pricefeed client for sending prices for the pricefeed server to consume. These prices
	// are retrieved via third-party APIs like Binance and then are encoded in-memory and
	// periodically sent via gRPC to a shared socket with the server.
	priceFeedClient := pricefeedclient.StartNewClient(
		// The client will use `context.Background` so that it can have a different context from
		// the main application.
		context.Background(),
		daemonFlags,
		grpcAddress,
		logger,
		&daemontypes.GrpcClientImpl{},
		marketParamsConfig,
		exchangeQueryConfig,
		constants.StaticExchangeDetails,
		&pricefeedclient.SubTaskRunnerImpl{},
	)

	RegisterDaemonWithHealthMonitor(priceFeedClient, daemonHealthMonitor, maxDaemonUnhealthyDuration, logger)

	go func() {
		reporterClient := reporterclient.NewClient(logger, cast.ToString(appOpts.Get("minimum-gas-prices")))
		if err := reporterClient.Start(
			context.Background(),
			daemonFlags,
			grpcAddress,
			&daemontypes.GrpcClientImpl{},
			marketParamsConfig,
			indexPriceCache,
			tokenDepositsCache,
			tokenBridgeTipsCache,
			queries,
			chainId,
		); err != nil {
			panic(err)
		}
	}()

	_ = tokenbridgeclient.StartNewClient(context.Background(), logger, tokenDepositsCache, tokenBridgeTipsCache)
	// }
	// Start the Metrics Daemon.
	// The metrics daemon is purely used for observability. It should never bring the app down.
	// Note: the metrics daemon is such a simple go-routine that we don't bother implementing a health-check
	// for this service. The task loop does not produce any errors because the telemetry calls themselves are
	// not error-returning, so in effect this daemon would never become unhealthy.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(
					"Metrics Daemon exited unexpectedly with a panic.",
					"panic",
					r,
					"stack",
					string(debug.Stack()),
				)
			}
		}()
		metricsclient.Start(
			// The client will use `context.Background` so that it can have a different context from
			// the main application.
			context.Background(),
			logger,
		)
	}()

	select {}
}

// RegisterDaemonWithHealthMonitor registers a daemon service with the update monitor, which will commence monitoring
// the health of the daemon. If the daemon does not register, the method will panic.
func RegisterDaemonWithHealthMonitor(
	healthCheckableDaemon daemontypes.HealthCheckable,
	daemonHealthMonitor *daemonservertypes.HealthMonitor,
	maxDaemonUnhealthyDuration time.Duration,
	logger log.Logger,
) {
	if err := daemonHealthMonitor.RegisterService(healthCheckableDaemon, maxDaemonUnhealthyDuration); err != nil {
		logger.Error(
			"Failed to register daemon service with update monitor",
			"error",
			err,
			"service",
			healthCheckableDaemon.ServiceName(),
			"maxDaemonUnhealthyDuration",
			maxDaemonUnhealthyDuration,
		)
		panic(err)
	}
}
