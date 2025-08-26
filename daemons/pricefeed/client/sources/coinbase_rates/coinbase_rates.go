package coinbase_rates

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	price_function "github.com/tellor-io/layer/daemons/pricefeed/client/sources"
	"github.com/tellor-io/layer/daemons/pricefeed/types"
)

type CoinbaseRatesResponse struct {
	Data CoinbaseRatesData `json:"data"`
}

type CoinbaseRatesData struct {
	Currency string            `json:"currency"`
	Rates    map[string]string `json:"rates"`
}

type CoinbaseRatesTicker struct {
	Pair  string
	Price string
}

var _ price_function.Ticker = (*CoinbaseRatesTicker)(nil)

func (t CoinbaseRatesTicker) GetPair() string {
	return t.Pair
}

func (t CoinbaseRatesTicker) GetAskPrice() string {
	return t.Price
}

func (t CoinbaseRatesTicker) GetBidPrice() string {
	return t.Price
}

func (t CoinbaseRatesTicker) GetLastPrice() string {
	return t.Price
}

func CoinbaseRatesPriceFunction(
	response *http.Response,
	tickerToExponent map[string]int32,
	resolver types.Resolver,
) (tickerToPrice map[string]uint64, unavailableTickers map[string]error, err error) {

	var ratesResponse CoinbaseRatesResponse
	err = json.NewDecoder(response.Body).Decode(&ratesResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode Coinbase rates response: %w", err)
	}

	var tickers []CoinbaseRatesTicker
	unavailableTickers = make(map[string]error)

	for ticker := range tickerToExponent {
		parts := strings.Split(ticker, "-")
		if len(parts) != 2 {
			unavailableTickers[ticker] = fmt.Errorf("invalid ticker format: %s", ticker)
			continue
		}

		baseCurrency := parts[0]
		quoteCurrency := parts[1]

		if quoteCurrency != "USD" {
			unavailableTickers[ticker] = fmt.Errorf("only USD quote currency is supported, got: %s", quoteCurrency)
			continue
		}

		rateStr, exists := ratesResponse.Data.Rates[baseCurrency]
		if !exists {
			unavailableTickers[ticker] = fmt.Errorf("rate not found for currency: %s", baseCurrency)
			continue
		}

		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			unavailableTickers[ticker] = fmt.Errorf("failed to parse rate for %s: %w", baseCurrency, err)
			continue
		}

		if rate == 0 {
			unavailableTickers[ticker] = fmt.Errorf("zero rate for currency: %s", baseCurrency)
			continue
		}
		price := 1.0 / rate

		tickers = append(tickers, CoinbaseRatesTicker{
			Pair:  ticker,
			Price: fmt.Sprintf("%f", price),
		})
	}

	tickerToPrice, unavailableFromHelper, err := price_function.GetMedianPricesFromTickers(
		tickers,
		tickerToExponent,
		resolver,
	)
	if err != nil {
		return nil, nil, err
	}

	for ticker, err := range unavailableFromHelper {
		if _, alreadySet := unavailableTickers[ticker]; !alreadySet {
			unavailableTickers[ticker] = err
		}
	}

	return tickerToPrice, unavailableTickers, nil
}
