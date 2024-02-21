package coingecko

import (
	"encoding/json"
	"net/http"

	price_function "github.com/tellor-io/layer/daemons/pricefeed/client/sources"
	"github.com/tellor-io/layer/daemons/pricefeed/types"
)

// CoingeckoTicker is our representation of ticker information returned in Coingecko response.
// Need to implement interface `Ticker` in util.go.
type CoingeckoTicker struct {
	Pair      string `json:"currency_pair" validate:"required"`
	AskPrice  string `json:"lowest_ask" validate:"required,positive-float-string"`
	BidPrice  string `json:"highest_bid" validate:"required,positive-float-string"`
	LastPrice string `json:"last" validate:"required,positive-float-string"`
}

// Ensure that GateTicker implements the Ticker interface at compile time.
var _ price_function.Ticker = (*CoingeckoTicker)(nil)

func (t CoingeckoTicker) GetPair() string {
	return t.Pair
}

func (t CoingeckoTicker) GetAskPrice() string {
	return t.AskPrice
}

func (t CoingeckoTicker) GetBidPrice() string {
	return t.BidPrice
}

func (t CoingeckoTicker) GetLastPrice() string {
	return t.LastPrice
}

// CoingeckoPriceFunction transforms an API response from Coingecko into a map of tickers to prices that have been
// shifted by a market specific exponent.
func CoingeckoPriceFunction(
	response *http.Response,
	tickerToExponent map[string]int32,
	resolver types.Resolver,
) (tickerToPrice map[string]uint64, unavailableTickers map[string]error, err error) {
	// Unmarshal response body into a list of tickers.
	var coingeckoTickers []CoingeckoTicker
	err = json.NewDecoder(response.Body).Decode(&coingeckoTickers)
	if err != nil {
		return nil, nil, err
	}

	return price_function.GetMedianPricesFromTickers(
		coingeckoTickers,
		tickerToExponent,
		resolver,
	)
}
