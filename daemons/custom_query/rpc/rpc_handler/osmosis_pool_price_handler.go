package rpchandler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

type OsmosisPoolPriceHandler struct{}

const (
	STATOM_ADDRESS = "ibc/C140AFD542AE77BD7DCC83F13FDD8C5E5BB8C4929785E6EC2F4C636F98F17901"
	ATOM_ADDRESS   = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
)

func (h *OsmosisPoolPriceHandler) FetchValue(
	ctx context.Context, reader *reader.Reader, _ bool, usdViaID uint32,
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
	if value == nil {
		return 0, fmt.Errorf("no value found at response path %v", reader.ResponsePath)
	}
	// value should be a dictionary with key "current_sqrt_price" and "token0" keys
	/*
			{
		  "pool": {
		    "@type": "/osmosis.concentratedliquidity.v1beta1.Pool",
		    "address": "osmo1n7cmdy4j3n7x24g547c6qd5crtdawephph3mq9a0dw8eh9fadszqu0uwvn",
		    "incentives_address": "osmo1h9qzx4rgzdexg39usx59h9plnn606mdgfk8aymk667s96hvte5tshp00xr",
		    "spread_rewards_address": "osmo1awtgwnc5wqqz6q2538ljyea5840c5c80pmy829klhqjkjaletnnqc53xpx",
		    "id": "1136",
		    "current_tick_liquidity": "1136084695464.541460080257116728",
		    "token0": "ibc/C140AFD542AE77BD7DCC83F13FDD8C5E5BB8C4929785E6EC2F4C636F98F17901",
		    "token1": "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
		    "current_sqrt_price": "1.290423192359603845831774199099128352",
		    "current_tick": "665192",
		    "tick_spacing": "100",
		    "exponent_at_price_one": "-6",
		    "spread_factor": "0.003000000000000000",
		    "last_liquidity_update": "2025-09-15T07:39:46.461182303Z"
		  }
		}
	*/
	data, ok := value.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("expected a dictionary for value, got %T", value)
	}
	currentSqrtPrice, ok := data["current_sqrt_price"]
	if !ok {
		return 0, fmt.Errorf("current_sqrt_price not found in JSON")
	}
	token0, ok := data["token0"].(string)
	if !ok {
		return 0, fmt.Errorf("token0 not found in JSON")
	}
	token1, ok := data["token1"].(string)
	if !ok {
		return 0, fmt.Errorf("token1 not found in JSON")
	}
	if !strings.EqualFold(token0, STATOM_ADDRESS) && !strings.EqualFold(token1, ATOM_ADDRESS) {
		return 0, errors.New("pool does not contain expected tokens")
	}
	var sqrtPrice float64
	switch v := currentSqrtPrice.(type) {
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
	usdViaParam, found := constants.StaticMarketParamsConfig[usdViaID]
	if !found {
		return 0, fmt.Errorf("market param not found for ID %d", usdViaID)
	}

	// Get usdVia price from cache
	usdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdViaParam}, time.Now())
	usdPriceRaw, found := usdPriceMap[usdViaID]
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
