package rpchandler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

type OsmosisPoolPriceHandler struct{}

func (h *OsmosisPoolPriceHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, _ bool, usdViaID int,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	resp, err := reader.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch JSON: %w", err)
	}
	value, err := reader.ExtractValueFromJSON(resp, reader.ResponsePath)
	if err != nil {
		return 0, fmt.Errorf("failed to extract value from JSON: %w", err)
	}

	var sqrtPrice float64
	switch v := value.(type) {
	case float64:
		sqrtPrice = v
	case string:
		sqrtPrice, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse sqrt price as float: %w", err)
		}
	default:
		return 0, fmt.Errorf("unexpected type for sqrt price: %T", value)
	}

	// Square the sqrt price to get the actual price
	currentPrice := sqrtPrice * sqrtPrice

	// Get parameter for usdViaID
	usdViaParam, found := constants.StaticMarketParamsConfig[uint32(usdViaID)]
	if !found {
		return 0, fmt.Errorf("market param not found for ID %d", usdViaID)
	}

	// Get usdVia price from cache
	usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
	usdPriceRaw, found := usdPriceMap[uint32(usdViaID)]
	if !found {
		return 0, errors.New("no valid USD via price found in cache")
	}

	usdPrice := float64(usdPriceRaw) * math.Pow10(int(usdViaParam.Exponent))

	// Return the final USD price
	finalPrice := usdPrice * currentPrice
	fmt.Printf("Osmosis pool price calculation: sqrtPrice=%.10f, price=%.10f, usdViaPrice=%.6f, finalPrice=%.6f\n",
		sqrtPrice, currentPrice, usdPrice, finalPrice)

	return finalPrice, nil
}
