package rpchandler

import (
	"context"
	"fmt"

	// "github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	// marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

type GenericHandler struct{}

func (h *GenericHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, invert bool, usdViaID int,
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

	switch v := current.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(v, "%f", &f)
		if err != nil {
			return 0, fmt.Errorf("error parsing string as float: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported value type: %T", current)
	}
}
