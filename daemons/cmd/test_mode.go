package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cosmossdk.io/log"
	"github.com/tellor-io/layer/daemons/configs"
	"github.com/tellor-io/layer/daemons/constants"
	customquery "github.com/tellor-io/layer/daemons/custom_query"
	"github.com/tellor-io/layer/daemons/lib"
	libtime "github.com/tellor-io/layer/daemons/lib/time"
	handler "github.com/tellor-io/layer/daemons/pricefeed/client/queryhandler"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
	daemontypes "github.com/tellor-io/layer/daemons/types"
)

// exchangeTestResult represents the result of testing an exchange for a market
type exchangeTestResult struct {
	Success bool
	Price   uint64
	Error   string
}

// runTestMode loads all price feed configurations and tests them
func runTestMode(homePath string, logger log.Logger) error {
	logger.Info("Starting test mode - verifying price feed configurations")

	// Load configurations
	logger.Info("Loading market parameters...")
	marketParams := configs.ReadMarketParamsConfigFile(homePath)
	logger.Info("Loaded market parameters", "count", len(marketParams))

	logger.Info("Loading exchange configurations...")
	exchangeConfigs := configs.ReadExchangeQueryConfigFile(homePath)
	logger.Info("Loaded exchange configurations", "count", len(exchangeConfigs))

	// Load custom queries
	logger.Info("Loading custom query configurations...")
	customQueries, err := customquery.BuildQueryEndpoints(homePath, "config", "custom_query_config.toml")
	if err != nil {
		logger.Warn("Failed to load custom queries (may not exist)", "error", err)
		customQueries = make(map[string]customquery.QueryConfig)
	} else {
		logger.Info("Loaded custom query configurations", "count", len(customQueries))
	}

	// Test each market param
	logger.Info("Testing price feeds...")
	for _, marketParam := range marketParams {
		if err := testMarketParam(marketParam, exchangeConfigs, logger); err != nil {
			logger.Error("Failed to test market param", "pair", marketParam.Pair, "id", marketParam.Id, "error", err)
		}
	}

	// Test custom queries
	if len(customQueries) > 0 {
		logger.Info("Testing custom queries...")
		for queryId, queryConfig := range customQueries {
			if err := testCustomQuery(queryId, queryConfig, logger); err != nil {
				logger.Error("Failed to test custom query", "query_id", queryId, "error", err)
			}
		}
	}

	logger.Info("Test mode completed successfully")
	return nil
}

// testMarketParam tests a single market param by querying all configured exchanges
func testMarketParam(
	marketParam types.MarketParam,
	exchangeConfigs map[types.ExchangeId]*types.ExchangeQueryConfig,
	logger log.Logger,
) error {
	logger.Info("Testing market", "pair", marketParam.Pair, "id", marketParam.Id)

	// Parse ExchangeConfigJson to get exchange ticker mappings
	var exchangeConfigJson types.ExchangeConfigJson
	if err := json.Unmarshal([]byte(marketParam.ExchangeConfigJson), &exchangeConfigJson); err != nil {
		return fmt.Errorf("failed to parse exchange config JSON: %w", err)
	}

	// Build market name to ID mapping for adjust-by-market support
	// For test mode, we'll skip adjust-by-market validation as it requires all market params
	marketNameToId := make(map[string]types.MarketId)
	marketNameToId[marketParam.Pair] = marketParam.Id

	// Track results for each exchange
	exchangeResults := make(map[types.ExchangeId]exchangeTestResult)
	var validPrices []uint64

	// Test each configured exchange
	for _, exchangeConfigJsonItem := range exchangeConfigJson.Exchanges {
		exchangeId := types.ExchangeId(exchangeConfigJsonItem.ExchangeName)

		// Check if exchange details exist
		exchangeDetails, exists := constants.StaticExchangeDetails[exchangeId]
		if !exists {
			exchangeResults[exchangeId] = exchangeTestResult{
				Success: false,
				Error:   "no exchange details found",
			}
			continue
		}

		// Check if exchange query config exists
		exchangeQueryConfig, hasConfig := exchangeConfigs[exchangeId]
		if !hasConfig {
			exchangeResults[exchangeId] = exchangeTestResult{
				Success: false,
				Error:   "exchange query config not found",
			}
			continue
		}

		// Query the exchange
		result := queryExchangeForMarket(
			exchangeId,
			exchangeDetails,
			*exchangeQueryConfig,
			marketParam,
			exchangeConfigJsonItem,
			logger,
		)
		exchangeResults[exchangeId] = result

		if result.Success {
			validPrices = append(validPrices, result.Price)
		}
	}

	// Calculate median
	var medianPrice uint64
	var medianErr error
	if len(validPrices) >= int(marketParam.MinExchanges) {
		medianPrice, medianErr = lib.Median[uint64](validPrices)
	} else {
		medianErr = fmt.Errorf("insufficient valid prices: got %d, need %d", len(validPrices), marketParam.MinExchanges)
	}

	// Log results
	logger.Info("Market test results",
		"pair", marketParam.Pair,
		"id", marketParam.Id,
		"valid_sources", len(validPrices),
		"total_sources", len(exchangeResults),
		"min_required", marketParam.MinExchanges,
		"median_price", medianPrice,
		"median_error", medianErr,
	)

	// Log individual exchange results
	for exchangeId, result := range exchangeResults {
		if result.Success {
			logger.Info("  ✓ Exchange succeeded",
				"exchange", exchangeId,
				"price", result.Price,
			)
		} else {
			logger.Warn("  ✗ Exchange failed",
				"exchange", exchangeId,
				"error", result.Error,
			)
		}
	}

	if medianErr != nil {
		return fmt.Errorf("failed to calculate median: %w", medianErr)
	}

	return nil
}

// queryExchangeForMarket queries a single exchange for a market and returns the result
func queryExchangeForMarket(
	exchangeId types.ExchangeId,
	exchangeDetails types.ExchangeQueryDetails,
	exchangeConfig types.ExchangeQueryConfig,
	marketParam types.MarketParam,
	exchangeConfigJsonItem types.ExchangeMarketConfigJson,
	logger log.Logger,
) exchangeTestResult {
	// Create a mutable exchange market config for this test
	mutableConfig := &types.MutableExchangeMarketConfig{
		Id:                   exchangeId,
		MarketToMarketConfig: make(map[types.MarketId]types.MarketConfig),
	}

	// Create market config from exchange config JSON
	marketConfig := types.MarketConfig{
		Ticker: exchangeConfigJsonItem.Ticker,
		Invert: exchangeConfigJsonItem.Invert,
	}

	// Handle AdjustByMarket if present (for test mode, we'll skip this as it requires
	// all market params to be loaded, which is complex for a simple test)
	if exchangeConfigJsonItem.AdjustByMarket != "" {
		// In a full implementation, we'd look up the market ID here
		// For test mode, we'll log a warning and continue
		logger.Debug("AdjustByMarket specified but not fully supported in test mode",
			"exchange", exchangeId,
			"adjust_by_market", exchangeConfigJsonItem.AdjustByMarket,
		)
	}

	mutableConfig.MarketToMarketConfig[marketParam.Id] = marketConfig

	// Create market price exponent map
	marketExponents := map[types.MarketId]types.Exponent{
		marketParam.Id: marketParam.Exponent,
	}

	// Create query handler
	queryHandler := &handler.ExchangeQueryHandlerImpl{
		TimeProvider: &libtime.TimeProviderImpl{},
	}

	// Create request handler with a new HTTP client
	httpClient := &http.Client{
		Timeout: time.Duration(exchangeConfig.TimeoutMs) * time.Millisecond,
	}
	requestHandler := daemontypes.NewRequestHandlerImpl(httpClient)

	// Query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(exchangeConfig.TimeoutMs)*time.Millisecond)
	defer cancel()

	marketIds := []types.MarketId{marketParam.Id}

	prices, unavailableMarkets, err := queryHandler.Query(
		ctx,
		&exchangeDetails,
		mutableConfig,
		marketIds,
		requestHandler,
		marketExponents,
	)

	if err != nil {
		return exchangeTestResult{
			Success: false,
			Error:   err.Error(),
		}
	}

	if len(unavailableMarkets) > 0 {
		if err, ok := unavailableMarkets[marketParam.Id]; ok {
			return exchangeTestResult{
				Success: false,
				Error:   fmt.Sprintf("market unavailable: %v", err),
			}
		}
	}

	if len(prices) == 0 {
		return exchangeTestResult{
			Success: false,
			Error:   "no prices returned",
		}
	}

	return exchangeTestResult{
		Success: true,
		Price:   prices[0].Price,
	}
}

// testCustomQuery tests a single custom query configuration
func testCustomQuery(queryId string, queryConfig customquery.QueryConfig, logger log.Logger) error {
	logger.Info("Testing custom query", "query_id", queryId)

	// Create an empty price cache for custom queries that may need it
	priceCache := pricefeedservertypes.NewMarketToExchangePrices(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := customquery.FetchPrice(ctx, queryConfig, priceCache)
	if err != nil {
		logger.Warn("  ✗ Custom query failed",
			"query_id", queryId,
			"error", err,
		)
		return err
	}

	logger.Info("  ✓ Custom query succeeded",
		"query_id", queryId,
		"encoded_value", results.EncodedValue,
	)

	return nil
}
