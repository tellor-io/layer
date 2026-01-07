package daemons

import (
	"context"
	"os"
	"runtime/debug"
	"sync"
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
	ctx                 context.Context
	wg                  sync.WaitGroup
	logger              log.Logger
	tempDir             string // Track temp directory for cleanup
}

// NewApp creates a new daemon application with the given context for graceful shutdown.
func NewApp(
	ctx context.Context,
	logger log.Logger,
	chainId,
	grpcAddress,
	homePath string,
	prometheusPort int,
) *App {
	appInstance := &App{
		ctx:    ctx,
		logger: logger,
	}

	RegisterTelemetryIfEnabled(logger, homePath, prometheusPort)
	// Create a temporary directory for app options (only the path string is used, not the actual directory)
	// This matches the pattern used in cmd/layerd/cmd/root.go
	tempDir, err := os.MkdirTemp("", "tellorapp")
	if err != nil {
		// Fallback to default if temp dir creation fails
		tempDir = app.DefaultNodeHome
	} else {
		appInstance.tempDir = tempDir // Track for cleanup
	}
	appOpts := simtestutil.NewAppOptionsWithFlagHome(tempDir)
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

	appInstance.Server = server

	server.WithPriceFeedMarketToExchangePrices(indexPriceCache)
	daemonHealthMonitor := daemonservertypes.NewHealthMonitor(
		daemonservertypes.DaemonStartupGracePeriod,
		daemonservertypes.HealthCheckPollFrequency,
		logger,
		daemonFlags.Shared.PanicOnDaemonFailureEnabled,
	)
	appInstance.DaemonHealthMonitor = daemonHealthMonitor

	server.WithTokenDepositsCache(tokenDepositsCache)
	// Create a closure for starting pricefeed daemon and daemon server. Daemon services are delayed until after the gRPC
	// service is started because daemons depend on the gRPC service being available. If a node is initialized
	// with a genesis time in the future, then the gRPC service will not be available until the genesis time, the
	// daemons will not be able to connect to the cosmos gRPC query service and finish initialization, and the daemon
	// monitoring service will panic.

	// set flag `--price-daemon-max-unhealthy-seconds=0` to disable
	maxDaemonUnhealthyDuration := time.Duration(daemonFlags.Shared.MaxDaemonUnhealthySeconds) * time.Second
	// Start server for handling gRPC messages from daemons.
	appInstance.wg.Add(1)
	go func() {
		defer appInstance.wg.Done()
		// Start server in a goroutine so we can monitor context
		done := make(chan struct{})
		go func() {
			defer close(done)
			server.Start()
		}()

		// Wait for either context cancellation or server completion
		select {
		case <-ctx.Done():
			// Context cancelled, stop the server gracefully
			logger.Info("Server: context cancelled, stopping server")
			server.Stop()
			// Wait for server to actually stop
			<-done
		case <-done:
			// Server stopped on its own (shouldn't happen normally)
		}
	}()

	exchangeQueryConfig := configs.ReadExchangeQueryConfigFile(homePath)
	marketParamsConfig := configs.ReadMarketParamsConfigFile(homePath)
	// Start pricefeed client for sending prices for the pricefeed server to consume. These prices
	// are retrieved via third-party APIs like Binance and then are encoded in-memory and
	// periodically sent via gRPC to a shared socket with the server.
	priceFeedClient := pricefeedclient.StartNewClient(
		// Use cancellable context instead of context.Background
		ctx,
		daemonFlags,
		grpcAddress,
		logger,
		&daemontypes.GrpcClientImpl{},
		marketParamsConfig,
		exchangeQueryConfig,
		constants.StaticExchangeDetails,
		&pricefeedclient.SubTaskRunnerImpl{},
	)
	appInstance.PriceFeedClient = priceFeedClient

	RegisterDaemonWithHealthMonitor(priceFeedClient, daemonHealthMonitor, maxDaemonUnhealthyDuration, logger)

	appInstance.wg.Add(1)
	go func() {
		defer appInstance.wg.Done()
		reporterClient := reporterclient.NewClient(logger, cast.ToString(appOpts.Get("minimum-gas-prices")))
		appInstance.ReporterClient = reporterClient
		if err := reporterClient.Start(
			// Use cancellable context instead of context.Background
			ctx,
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
			logger.Error("Reporter client failed to start", "error", err)
		}
	}()

	tokenBridgeClient := tokenbridgeclient.StartNewClient(ctx, logger, tokenDepositsCache, tokenBridgeTipsCache)
	appInstance.TokenBridgeClient = tokenBridgeClient

	// Start the Metrics Daemon.
	// The metrics daemon is purely used for observability. It should never bring the app down.
	// Note: the metrics daemon is such a simple go-routine that we don't bother implementing a health-check
	// for this service. The task loop does not produce any errors because the telemetry calls themselves are
	// not error-returning, so in effect this daemon would never become unhealthy.
	appInstance.wg.Add(1)
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
			appInstance.wg.Done()
		}()
		metricsclient.Start(
			// Use cancellable context instead of context.Background
			ctx,
			logger,
		)
	}()

	return appInstance
}

// Shutdown gracefully shuts down all daemon services
func (a *App) Shutdown() {
	a.logger.Info("Initiating graceful shutdown...")

	// Stop all clients
	if a.PriceFeedClient != nil {
		a.logger.Info("Stopping pricefeed client...")
		a.PriceFeedClient.Stop()
		a.logger.Info("Pricefeed client stopped")
	}

	if a.TokenBridgeClient != nil {
		a.logger.Info("Stopping token bridge client...")
		a.TokenBridgeClient.Stop()
		a.logger.Info("Token bridge client stopped")
	}

	if a.ReporterClient != nil {
		a.logger.Info("Stopping reporter client...")
		a.ReporterClient.Stop()
		a.logger.Info("Reporter client stopped")
	}

	if a.DaemonHealthMonitor != nil {
		a.logger.Info("Stopping health monitor...")
		a.DaemonHealthMonitor.Stop()
		a.logger.Info("Health monitor stopped")
	}

	// Stop gRPC server
	if a.Server != nil {
		a.logger.Info("Stopping gRPC server...")
		a.Server.Stop()
		a.logger.Info("gRPC server stopped")
	}

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed successfully
	case <-time.After(45 * time.Second):
		a.logger.Error("Timeout waiting for goroutines to stop")
	}

	// Clean up temporary directory if one was created
	if a.tempDir != "" && a.tempDir != app.DefaultNodeHome {
		if err := os.RemoveAll(a.tempDir); err != nil {
			a.logger.Error("Failed to remove temporary directory", "path", a.tempDir, "error", err)
		} else {
			a.logger.Info("Temporary directory cleaned up", "path", a.tempDir)
		}
	}

	a.logger.Info("App shutdown complete")
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
