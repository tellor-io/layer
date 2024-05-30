package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	"github.com/tellor-io/layer/daemons/pricefeed/client/price_encoder"
	"github.com/tellor-io/layer/daemons/pricefeed/client/price_fetcher"
	handler "github.com/tellor-io/layer/daemons/pricefeed/client/queryhandler"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	servertypes "github.com/tellor-io/layer/daemons/server/types"
	daemontypes "github.com/tellor-io/layer/daemons/types"
	"github.com/tellor-io/layer/lib/metrics"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

var HttpClient = http.Client{
	Transport: &http.Transport{MaxConnsPerHost: constants.MaxConnectionsPerExchange},
}

// SubTaskRunnerImpl is the struct that implements the `SubTaskRunner` interface.
type SubTaskRunnerImpl struct{}

// Ensure the `SubTaskRunnerImpl` struct is implemented at compile time.
var _ SubTaskRunner = (*SubTaskRunnerImpl)(nil)

// SubTaskRunner is the interface for running pricefeed client task functions.
type SubTaskRunner interface {
	StartPriceUpdater(
		c *Client,
		ctx context.Context,
		ticker *time.Ticker,
		stop <-chan bool,
		exchangeToMarketPrices types.ExchangeToMarketPrices,
		priceFeedServiceClient servertypes.PriceFeedServiceClient,
		logger log.Logger,
	)
	StartPriceEncoder(
		exchangeId types.ExchangeId,
		configs types.PricefeedMutableMarketConfigs,
		exchangeToMarketPrices types.ExchangeToMarketPrices,
		logger log.Logger,
		bCh <-chan *price_fetcher.PriceFetcherSubtaskResponse,
	)
	StartPriceFetcher(
		ticker *time.Ticker,
		stop <-chan bool,
		configs types.PricefeedMutableMarketConfigs,
		exchangeQueryConfig types.ExchangeQueryConfig,
		exchangeDetails types.ExchangeQueryDetails,
		queryHandler handler.ExchangeQueryHandler,
		logger log.Logger,
		bCh chan<- *price_fetcher.PriceFetcherSubtaskResponse,
	)
}

// StartPriceEncoder continuously reads from a buffered channel, reading encoded API responses for exchange
// requests and inserting them into an `ExchangeToMarketPrices` cache, performing currency conversions based
// on the index price of other markets as necessary.
// StartPriceEncoder reads price fetcher responses from a shared channel, and does not need a ticker or stop
// signal from the daemon to exit. It marks itself as done in the daemon's wait group when the price fetcher
// closes the shared channel.
func (s *SubTaskRunnerImpl) StartPriceEncoder(
	exchangeId types.ExchangeId,
	configs types.PricefeedMutableMarketConfigs,
	exchangeToMarketPrices types.ExchangeToMarketPrices,
	logger log.Logger,
	bCh <-chan *price_fetcher.PriceFetcherSubtaskResponse,
) {
	exchangeMarketConfig, err := configs.GetExchangeMarketConfigCopy(exchangeId)
	if err != nil {
		panic(err)
	}

	marketConfigs, err := configs.GetMarketConfigCopies(exchangeMarketConfig.GetMarketIds())
	if err != nil {
		panic(err)
	}

	priceEncoder, err := price_encoder.NewPriceEncoder(
		exchangeMarketConfig,
		marketConfigs,
		exchangeToMarketPrices,
		logger,
		bCh,
	)
	if err != nil {
		panic(err)
	}

	configs.AddPriceEncoder(priceEncoder)

	// Listen for prices from the buffered channel and update the exchangeToMarketPrices cache.
	// Also log any errors that occur.
	for response := range bCh {
		priceEncoder.ProcessPriceFetcherResponse(response)
	}
}

// StartPriceFetcher periodically starts goroutines to "fetch" market prices from a specific exchange. Each
// goroutine does the following:
// 1) query a single market price from a specific exchange
// 2) transform response to `MarketPriceTimestamp`
// 3) send transformed response to a buffered channel that's shared across multiple goroutines
// NOTE: the subtask response shared channel has a buffer size and goroutines will block if the buffer is full.
// NOTE: the price fetcher kicks off 1 to n go routines every time the subtask loop runs, but the subtask
// loop blocks until all go routines are done. This means that these go routines are not tracked by the wait group.
func (s *SubTaskRunnerImpl) StartPriceFetcher(
	ticker *time.Ticker,
	stop <-chan bool,
	configs types.PricefeedMutableMarketConfigs,
	exchangeQueryConfig types.ExchangeQueryConfig,
	exchangeDetails types.ExchangeQueryDetails,
	queryHandler handler.ExchangeQueryHandler,
	logger log.Logger,
	bCh chan<- *price_fetcher.PriceFetcherSubtaskResponse,
) {
	exchangeMarketConfig, err := configs.GetExchangeMarketConfigCopy(exchangeQueryConfig.ExchangeId)
	if err != nil {
		panic(err)
	}

	marketConfigs, err := configs.GetMarketConfigCopies(exchangeMarketConfig.GetMarketIds())
	if err != nil {
		panic(err)
	}

	// Create PriceFetcher to begin querying with.
	priceFetcher, err := price_fetcher.NewPriceFetcher(
		exchangeQueryConfig,
		exchangeDetails,
		exchangeMarketConfig,
		marketConfigs,
		queryHandler,
		logger,
		bCh,
	)
	if err != nil {
		panic(err)
	}

	// The PricefeedMutableMarketConfigs object that stores the configs for each exchange
	// is not initialized with the price fetcher, because both objects have references to
	// each other defined during normal daemon operation. Instead, the price fetcher is
	// initialized with the configs object after the price fetcher is created, and then adds
	// itself to the config's list of exchange config updaters here.
	configs.AddPriceFetcher(priceFetcher)

	requestHandler := daemontypes.NewRequestHandlerImpl(
		&HttpClient,
	)
	// Begin loop to periodically start goroutines to query market prices.
	for {
		select {
		case <-ticker.C:
			// Start goroutines to query exchange markets. The goroutines started by the price
			// fetcher are not tracked by the global wait group, because RunTaskLoop will
			// block until all goroutines are done.
			priceFetcher.RunTaskLoop(requestHandler)

		case <-stop:
			// Signal to the encoder that the price fetcher is done.
			close(bCh)
			return
		}
	}
}

// StartPriceUpdater periodically runs a task loop to send price updates to the pricefeed server
// via:
// 1) Get `MarketPriceTimestamps` for all exchanges in an `ExchangeToMarketPrices` struct.
// 2) Transform `MarketPriceTimestamps` and exchange ids into an `UpdateMarketPricesRequest` struct.
// StartPriceUpdater runs in the daemon's main goroutine and does not need access to the daemon's wait group
// to signal task completion.
func (s *SubTaskRunnerImpl) StartPriceUpdater(
	c *Client,
	ctx context.Context,
	ticker *time.Ticker,
	stop <-chan bool,
	exchangeToMarketPrices types.ExchangeToMarketPrices,
	priceFeedServiceClient servertypes.PriceFeedServiceClient,
	logger log.Logger,
) {
	for {
		select {
		case <-ticker.C:
			err := RunPriceUpdaterTaskLoop(ctx, exchangeToMarketPrices, priceFeedServiceClient, logger)

			if err == nil {
				// Record update success for the daemon health check.
				c.ReportSuccess()
			} else {
				logger.Error("Failed to run price updater task loop for price daemon", constants.ErrorLogKey, err)
				// Record update failure for the daemon health check.
				c.ReportFailure(fmt.Errorf("failed to run price updater task loop for price daemon; %w", err))
			}

		case <-stop:
			return
		}
	}
}

// -------------------- Task Loops -------------------- //

// RunPriceUpdaterTaskLoop copies the map of current `exchangeId -> MarketPriceTimestamp`,
// transforms the map values into a market price update request and sends the request to the socket
// where the pricefeed server is listening.
func RunPriceUpdaterTaskLoop(
	ctx context.Context,
	exchangeToMarketPrices types.ExchangeToMarketPrices,
	priceFeedServiceClient servertypes.PriceFeedServiceClient,
	logger log.Logger,
) error {
	logger = logger.With(constants.SubmoduleLogKey, constants.PriceUpdaterSubmoduleName)
	priceUpdates := exchangeToMarketPrices.GetAllPrices()
	request := transformPriceUpdates(priceUpdates)

	// Measure latency to send prices over gRPC.
	// Note: intentionally skipping latency for `GetAllPrices`.
	defer telemetry.ModuleMeasureSince(
		metrics.PricefeedDaemon,
		time.Now(),
		metrics.PriceUpdaterSendPrices,
		metrics.Latency,
	)

	// On startup the length of request will likely be 0. Even so, we return an error here because this
	// is unexpected behavior once the daemon reaches a steady state. The daemon health check process should
	// be robust enough to ignore temporarily unhealthy daemons.
	// Sending a request of length 0, however, causes a panic.
	// panic: rpc error: code = Unknown desc = Market price update has length of 0.
	if len(request.MarketPriceUpdates) > 0 {
		_, err := priceFeedServiceClient.UpdateMarketPrices(ctx, request)
		if err != nil {
			// Log error if an error will be returned from the task loop and measure failure.
			logger.Error("Failed to run price updater task loop for price daemon", "error", err)
			telemetry.IncrCounter(
				1,
				metrics.PricefeedDaemon,
				metrics.PriceUpdaterTaskLoop,
				metrics.Error,
			)
			return err
		}
	} else {
		// This is expected to happen on startup until prices have been encoded into the in-memory
		// `exchangeToMarketPrices` map. After that point, there should be no price updates of length 0.
		logger.Info("Price update had length of 0")
		telemetry.IncrCounter(
			1,
			metrics.PricefeedDaemon,
			metrics.PriceUpdaterZeroPrices,
			metrics.Count,
		)
		return fmt.Errorf("ErrEmptyMarketPriceUpdate")
	}

	return nil
}

// transformPriceUpdates transforms a map (key: exchangeId, value: list of market prices) into a
// market price update request.
func transformPriceUpdates(
	updates map[types.ExchangeId][]types.MarketPriceTimestamp,
) *servertypes.UpdateMarketPricesRequest {
	// Measure latency to transform prices being sent over gRPC.
	defer telemetry.ModuleMeasureSince(
		metrics.PricefeedDaemon,
		time.Now(),
		metrics.PriceUpdaterTransformPrices,
		metrics.Latency,
	)

	marketPriceUpdateMap := make(map[types.MarketId]*servertypes.MarketPriceUpdate)

	// Invert to marketId -> `api.MarketPriceUpdate`.
	for exchangeId, marketPriceTimestamps := range updates {
		for _, marketPriceTimestamp := range marketPriceTimestamps {
			marketPriceUpdate, exists := marketPriceUpdateMap[marketPriceTimestamp.MarketId]
			// Add key with empty `api.MarketPriceUpdate` if entry does not exist.
			if !exists {
				marketPriceUpdate = &servertypes.MarketPriceUpdate{
					MarketId:       marketPriceTimestamp.MarketId,
					ExchangePrices: []*servertypes.ExchangePrice{},
				}
				marketPriceUpdateMap[marketPriceTimestamp.MarketId] = marketPriceUpdate
			}

			// Add `api.ExchangePrice`.
			priceUpdateTime := marketPriceTimestamp.LastUpdatedAt
			exchangePrice := &servertypes.ExchangePrice{
				ExchangeId:     exchangeId,
				Price:          marketPriceTimestamp.Price,
				LastUpdateTime: &priceUpdateTime,
			}
			marketPriceUpdate.ExchangePrices = append(marketPriceUpdate.ExchangePrices, exchangePrice)
		}
	}

	// Add all `api.MarketPriceUpdate` to request to be sent by `client.UpdateMarketPrices`.
	request := &servertypes.UpdateMarketPricesRequest{
		MarketPriceUpdates: make([]*servertypes.MarketPriceUpdate, 0, len(marketPriceUpdateMap)),
	}
	for _, update := range marketPriceUpdateMap {
		request.MarketPriceUpdates = append(
			request.MarketPriceUpdates,
			update,
		)
	}
	return request
}
