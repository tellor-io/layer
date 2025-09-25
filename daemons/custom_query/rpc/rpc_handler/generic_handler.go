package rpchandler

import (
	"context"
	"fmt"

	"math"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

type GenericHandler struct{}

func (h *GenericHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, invert bool, usdViaID uint32,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	resp, err := reader.FetchJSON(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch JSON: %w", err)
	}
	current, err := reader.ExtractValueFromJSON(resp, reader.ResponsePath)
	if err != nil {
		return 0, fmt.Errorf("failed to extract value from JSON: %w", err)
	}
	var value float64
	switch v := current.(type) {
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	case string:
		_, err := fmt.Sscanf(v, "%f", &value)
		if err != nil {
			return 0, fmt.Errorf("error parsing string as float: %w", err)
		}
	default:
		return 0, fmt.Errorf("unsupported value type: %T", current)
	}
	// Apply inversion if needed
	if invert {
		if value == 0 {
			return 0, fmt.Errorf("cannot invert zero value")
		}
		value = 1.0 / value
	}

	// Apply USD conversion via another market if specified
	if usdViaID != 0 {
		usdViaParam, found := constants.StaticMarketParamsConfig[usdViaID]
		if !found {
			return 0, fmt.Errorf("market param not found for ID %d", usdViaID)
		}

		// Get usdVia price from cache
		usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
		usdPriceRaw, found := usdPriceMap[usdViaID]
		if !found {
			return 0, fmt.Errorf("no valid USD via price found in cache for market ID %d", usdViaID)
		}

		// Convert raw price to float using the market's exponent
		usdPrice := float64(usdPriceRaw) * math.Pow10(int(usdViaParam.Exponent))

		// Multiply the value by the USD price
		value = value * usdPrice
	}

	return value, nil
}
