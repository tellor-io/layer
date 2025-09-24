package mexc

import (
	"encoding/json"
	"net/http"

	price_function "github.com/tellor-io/layer/daemons/pricefeed/client/sources"
	"github.com/tellor-io/layer/daemons/pricefeed/types"
)

// MexcTicker is our representation of ticker information returned in Mexc response.
// MexcTicker implements interface `Ticker` in util.go.
type MexcTicker struct {
	Pair      string `json:"symbol" validate:"required"`
	AskPrice  string `json:"askPrice" validate:"required,positive-float-string"`
	BidPrice  string `json:"bidPrice" validate:"required,positive-float-string"`
	LastPrice string `json:"lastPrice" validate:"required,positive-float-string"`
}

// Ensure that MexcTicker implements the Ticker interface at compile time.
var _ price_function.Ticker = (*MexcTicker)(nil)

func (t MexcTicker) GetPair() string {
	return t.Pair
}

func (t MexcTicker) GetAskPrice() string {
	return t.AskPrice
}

func (t MexcTicker) GetBidPrice() string {
	return t.BidPrice
}

func (t MexcTicker) GetLastPrice() string {
	return t.LastPrice
}

// MexcPriceFunction transforms an API response from Mexc into a map of tickers to prices that have been
// shifted by a market specific exponent.
func MexcPriceFunction(
	response *http.Response,
	tickerToExponent map[string]int32,
	resolver types.Resolver,
) (tickerToPrice map[string]uint64, unavailableTickers map[string]error, err error) {
	// Unmarshal response body.
	// The API now returns a direct array of tickers
	var tickers []MexcTicker
	err = json.NewDecoder(response.Body).Decode(&tickers)
	if err != nil {
		return nil, nil, err
	}

	return price_function.GetMedianPricesFromTickers(
		tickers,
		tickerToExponent,
		resolver,
	)
}
