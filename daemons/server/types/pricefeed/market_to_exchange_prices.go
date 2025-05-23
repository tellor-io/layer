package types

import (
	"sync"
	"time"

	gometrics "github.com/hashicorp/go-metrics"
	"github.com/tellor-io/layer/daemons/lib"
	"github.com/tellor-io/layer/daemons/lib/metrics"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedmetrics "github.com/tellor-io/layer/daemons/pricefeed/metrics"
	servertypes "github.com/tellor-io/layer/daemons/server/types/daemons"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

// MarketToExchangePrices maintains price info for multiple markets. Each
// market can support prices from multiple exchange sources. Specifically,
// MarketToExchangePrices supports methods to update prices and to retrieve
// median prices. Methods are goroutine safe.
type MarketToExchangePrices struct {
	sync.Mutex                                         // lock
	marketToExchangePrices map[uint32]*ExchangeToPrice // {k: market id, v: exchange prices}
	// maxPriceAge is the maximum age of a price before it is considered too stale to be used.
	// Prices older than this age will not be used to calculate the median price.
	maxPriceAge time.Duration
}

// NewMarketToExchangePrices creates a new MarketToExchangePrices.
func NewMarketToExchangePrices(maxPriceAge time.Duration) *MarketToExchangePrices {
	return &MarketToExchangePrices{
		marketToExchangePrices: make(map[uint32]*ExchangeToPrice),
		maxPriceAge:            maxPriceAge,
	}
}

// UpdatePrices updates market prices given a list of price updates. Prices are
// only updated if the timestamp on the updates are greater than the timestamp
// on existing prices.
func (mte *MarketToExchangePrices) UpdatePrices(
	updates []*servertypes.MarketPriceUpdate,
) {
	mte.Lock()
	defer mte.Unlock()
	for _, marketPriceUpdate := range updates {
		marketId := marketPriceUpdate.MarketId
		exchangeToPrices, ok := mte.marketToExchangePrices[marketId]
		if !ok {
			exchangeToPrices = NewExchangeToPrice(marketId)
			mte.marketToExchangePrices[marketId] = exchangeToPrices
		}
		exchangeToPrices.UpdatePrices(marketPriceUpdate.ExchangePrices)
	}
}

// GetValidMedianPrices returns median prices for multiple markets.
// Specifically, it returns a map where the key is the market ID and the value
// is the median price for the market. It only returns "valid" prices where
// a price is valid iff
// 1) the last update time is within a predefined threshold away from the given
// read time.
// 2) the number of prices that meet 1) are greater than the minimum number of
// exchanges specified in the given input.
func (mte *MarketToExchangePrices) GetValidMedianPrices(
	marketParams []types.MarketParam,
	readTime time.Time,
) map[uint32]uint64 {
	cutoffTime := readTime.Add(-mte.maxPriceAge)
	marketIdToMedianPrice := make(map[uint32]uint64)

	mte.Lock()
	defer mte.Unlock()
	for _, marketParam := range marketParams {
		marketId := marketParam.Id
		exchangeToPrice, ok := mte.marketToExchangePrices[marketId]
		if !ok {
			// No market price info yet, skip this market.
			telemetry.IncrCounterWithLabels(
				[]string{
					metrics.PricefeedServer,
					metrics.NoMarketPrice,
					metrics.Count,
				},
				1,
				[]gometrics.Label{
					pricefeedmetrics.GetLabelForMarketId(marketId),
				},
			)
			continue
		}

		// GetValidPriceForMarket filters prices based on cutoff time.
		validPrices := exchangeToPrice.GetValidPrices(cutoffTime)
		telemetry.SetGaugeWithLabels(
			[]string{
				metrics.PricefeedServer,
				metrics.ValidPrices,
				metrics.Count,
			},
			float32(len(validPrices)),
			[]gometrics.Label{
				pricefeedmetrics.GetLabelForMarketId(marketId),
			},
		)

		// The number of valid prices must be >= min number of exchanges.
		if len(validPrices) >= int(marketParam.MinExchanges) {
			// Calculate the median. Returns an error if the input is empty.
			median, err := lib.Median(validPrices)
			if err != nil {
				telemetry.IncrCounterWithLabels(
					[]string{
						metrics.PricefeedServer,
						metrics.NoValidMedianPrice,
						metrics.Count,
					},
					1,
					[]gometrics.Label{
						pricefeedmetrics.GetLabelForMarketId(marketId),
					},
				)
				continue
			}
			marketIdToMedianPrice[marketId] = median
		}
	}

	return marketIdToMedianPrice
}
