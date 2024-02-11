package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	daemontypes "github.com/tellor-io/layer/daemons/types"

	price_function "github.com/tellor-io/layer/daemons/pricefeed/client/sources"
	clienttypes "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/lib"
	libtime "github.com/tellor-io/layer/lib/time"
)

const (
	UnexpectedResponseStatusMessage = "Unexpected response status code of:"
)

var (
	RateLimitingError = fmt.Errorf("status 429 - rate limit exceeded")
)

// ExchangeQueryHandlerImpl is the struct that implements the `ExchangeQueryHandler` interface.
type ExchangeQueryHandlerImpl struct {
	libtime.TimeProvider
}

// Ensure the `ExchangeQueryHandlerImpl` struct is implemented at compile time
var _ ExchangeQueryHandler = (*ExchangeQueryHandlerImpl)(nil)

// ExchangeQueryHandler is an interface that encapsulates querying an exchange for price info.
type ExchangeQueryHandler interface {
	libtime.TimeProvider
	Query(
		ctx context.Context,
		exchangeQueryDetails *clienttypes.ExchangeQueryDetails,
		exchangeConfig *clienttypes.MutableExchangeMarketConfig,
		marketIds []clienttypes.MarketId,
		requestHandler daemontypes.RequestHandler,
		marketPriceExponent map[clienttypes.MarketId]clienttypes.Exponent,
	) (marketPriceTimestamps []*clienttypes.MarketPriceTimestamp, unavailableMarkets map[clienttypes.MarketId]error, err error)
}

// Query makes an API call to a specific exchange and returns the transformed response, including both valid prices
// and any unavailable markets with specific errors.
// 1) Validate `marketIds` contains at least one id.
// 2) Convert the list of `marketIds` to tickers that are specific for a given exchange. Create a mapping of
// tickers to price exponents and a reverse mapping of ticker back to `MarketId`.
// 3) Make API call to an exchange and verify the response status code is not an error status code.
// 4) Transform the API response to market prices, while tracking unavailable tickers.
// 5) Return dual values:
// - a slice of `MarketPriceTimestamp`s that contains resolved market prices
// - a map of marketIds that could not be resolved with corresponding specific errors.
func (eqh *ExchangeQueryHandlerImpl) Query(
	ctx context.Context,
	exchangeQueryDetails *clienttypes.ExchangeQueryDetails,
	exchangeConfig *clienttypes.MutableExchangeMarketConfig,
	marketIds []clienttypes.MarketId,
	requestHandler daemontypes.RequestHandler,
	marketPriceExponent map[clienttypes.MarketId]clienttypes.Exponent,
) (marketPriceTimestamps []*clienttypes.MarketPriceTimestamp, unavailableMarkets map[clienttypes.MarketId]error, err error) {
	// 1) Validate `marketIds` contains at least one id.
	if len(marketIds) == 0 {
		return nil, nil, errors.New("At least one marketId must be queried")
	}
	// 2) Convert the list of `marketIds` to tickers that are specific for a given exchange. Create a mapping
	// of tickers to price exponents and a reverse mapping of ticker back to `MarketId`.
	tickers := make([]string, 0, len(marketIds))
	tickerToPriceExponent := make(map[string]int32, len(marketIds))
	tickerToMarketId := make(map[string]clienttypes.MarketId, len(marketIds))
	for _, marketId := range marketIds {
		config, ok := exchangeConfig.MarketToMarketConfig[marketId]
		if !ok {
			return nil, nil, fmt.Errorf("No market config for market: %v", marketId)
		}
		priceExponent, ok := marketPriceExponent[marketId]
		if !ok {
			return nil, nil, fmt.Errorf("No market price exponent for id: %v", marketId)
		}

		tickers = append(tickers, config.Ticker)
		tickerToPriceExponent[config.Ticker] = priceExponent
		tickerToMarketId[config.Ticker] = marketId

	}

	// 3) Make API call to an exchange and verify the response status code is not an error status code.
	var url string
	if exchangeQueryDetails.Exchange == "Coingecko" {
		url = CreateCoingeckoRequestUrl(exchangeQueryDetails.Url, tickers)
	} else {
		url = CreateRequestUrl(exchangeQueryDetails.Url, tickers)
	}
	response, err := requestHandler.Get(ctx, url)
	if err != nil {
		return nil, nil, err
	}

	if response.StatusCode == 429 {
		return nil, nil, RateLimitingError
	}

	// Verify response is not 4xx or 5xx.
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, nil, fmt.Errorf("%s %v", UnexpectedResponseStatusMessage, response.StatusCode)
	}

	// 4) Transform the API response to market prices, while tracking unavailable tickers.
	prices, unavailableTickers, err := exchangeQueryDetails.PriceFunction(
		response,
		tickerToPriceExponent,
		lib.Median[uint64],
	)
	if err != nil {
		return nil, nil, price_function.NewExchangeError(exchangeQueryDetails.Exchange, err.Error())
	}

	// 5) Insert prices into MarketPriceTimestamp struct slice, convert unavailable tickers back into marketIds,
	// and return.
	marketPriceTimestamps = make([]*clienttypes.MarketPriceTimestamp, 0, len(prices))
	now := eqh.Now()

	for ticker, price := range prices {
		marketId, ok := tickerToMarketId[ticker]
		if !ok {
			return nil, nil, fmt.Errorf("Severe unexpected error: no market id for ticker: %v", ticker)
		}

		marketPriceTimestamp := &clienttypes.MarketPriceTimestamp{
			MarketId:      marketId,
			Price:         price,
			LastUpdatedAt: now,
		}

		marketPriceTimestamps = append(marketPriceTimestamps, marketPriceTimestamp)
	}

	unavailableMarkets = make(map[clienttypes.MarketId]error, len(unavailableTickers))
	for ticker, error := range unavailableTickers {
		marketId, ok := tickerToMarketId[ticker]
		if !ok {
			return nil, nil, fmt.Errorf("Severe unexpected error: no market id for ticker: %v", ticker)
		}
		unavailableMarkets[marketId] = error
	}

	return marketPriceTimestamps, unavailableMarkets, nil
}

func CreateRequestUrl(baseUrl string, tickers []string) string {
	return strings.Replace(baseUrl, "$", strings.Join(tickers, ","), -1)
}

func CreateCoingeckoRequestUrl(baseUrl string, tickers []string) string {
	if len(tickers) > 0 {
		parts := strings.Split(tickers[0], "-")
		if len(parts) == 2 {
			id := parts[0]
			currency := parts[1]
			urlWithId := strings.Replace(baseUrl, "ids=$", "ids="+id, 1)
			finalUrl := strings.Replace(urlWithId, "vs_currencies=$", "vs_currencies="+currency, 1)
			return finalUrl
		}
	}
	return baseUrl
}
