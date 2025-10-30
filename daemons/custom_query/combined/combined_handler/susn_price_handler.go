package combined_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

const STAKED_USN_CONTRACT = "0xe24a3dc889621612422a64e6388927901608b91d"

// SUSNPriceHandler calculates SUSN price by multiplying the contract conversion rate with an RPC price
type SUSNPriceHandler struct{}

func init() {
	RegisterHandler("susn_price", &SUSNPriceHandler{})
}

func (h *SUSNPriceHandler) FetchValue(
	ctx context.Context,
	contractReaders map[string]*contractreader.Reader,
	rpcReaders map[string]*rpcreader.Reader,
	_ *pricefeedservertypes.MarketToExchangePrices,
	minResponses int,
	maxSpreadPercent float64,
) (float64, error) {
	// Validate we have the required readers
	contractReader, exists := contractReaders["ethereum"]
	if !exists {
		return 0, fmt.Errorf("ethereum contract reader not found")
	}

	if len(rpcReaders) == 0 {
		return 0, fmt.Errorf("no RPC readers found")
	}

	fetcher := NewParallelFetcher()

	fetcher.FetchContract(
		ctx,
		"conversion_rate",
		contractReader,
		STAKED_USN_CONTRACT,
		"convertToAssets(uint256) returns (uint256)",
		[]string{"1000000000000000000"},
	)

	// Fetch prices from all RPC endpoints
	for name, reader := range rpcReaders {
		fetcher.FetchRPC(ctx, "price_"+name, reader)
	}

	// Wait for all fetches to complete
	fetcher.Wait()

	conversionBytes, err := fetcher.GetContractBytes("conversion_rate")
	if err != nil {
		return 0, fmt.Errorf("failed to get conversion rate: %w", err)
	}

	valueInUsdWei := new(big.Int).SetBytes(conversionBytes)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	divisorFloat := new(big.Float).SetInt(divisor)
	valueInUsdFloat := new(big.Float).SetInt(valueInUsdWei)
	conversionRate := new(big.Float).Quo(valueInUsdFloat, divisorFloat)
	conversionRateFloat, _ := conversionRate.Float64()

	var prices []float64
	for name, reader := range rpcReaders {
		priceBytes, err := fetcher.GetBytes("price_" + name)
		if err != nil {
			fmt.Printf("Warning: failed to get price from %s: %v\n", name, err)
			continue
		}

		price, err := h.extractPrice(priceBytes, reader.ResponsePath)
		if err != nil {
			fmt.Printf("Warning: failed to extract price from %s: %v\n", name, err)
			continue
		}

		prices = append(prices, price)
	}

	if len(prices) < minResponses {
		return 0, fmt.Errorf("insufficient SUSN/USD prices: got %d, need at least %d", len(prices), minResponses)
	}

	// pick out the min and max prices to calculate spread
	minPrice := prices[0]
	maxPrice := prices[0]
	for _, p := range prices {
		if p < minPrice {
			minPrice = p
		}
		if p > maxPrice {
			maxPrice = p
		}
	}

	if minPrice > 0 {
		spreadPercent := ((maxPrice - minPrice) / minPrice) * 100
		if spreadPercent > maxSpreadPercent {
			return 0, fmt.Errorf("SUSN/USD price spread of %.2f%% exceeds maximum allowed %.2f%%", spreadPercent, maxSpreadPercent)
		}
	}

	// Calculate median price
	medianPrice := h.calculateMedian(prices)

	// Multiply conversion rate by median price
	result := conversionRateFloat * medianPrice

	fmt.Printf("SUSN Price: conversion_rate=%f, median_price=%f, final=%f\n", conversionRateFloat, medianPrice, result)

	return result, nil
}

func (h *SUSNPriceHandler) calculateMedian(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	if len(prices) == 1 {
		return prices[0]
	}

	// Sort the prices
	sort.Float64s(prices)

	// Calculate median
	n := len(prices)
	if n%2 == 0 {
		return (prices[n/2-1] + prices[n/2]) / 2.0
	}
	// Odd number of elements: middle value
	return prices[n/2]
}

func (h *SUSNPriceHandler) extractPrice(data []byte, path []string) (float64, error) {
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	current := result
	for i, key := range path {
		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[key]
			if !ok {
				return 0, fmt.Errorf("key not found at path segment %d: %s", i, key)
			}
		default:
			return 0, fmt.Errorf("unexpected type at key %s: %T", key, current)
		}
	}

	// Convert to float64
	switch v := current.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err != nil {
			return 0, fmt.Errorf("failed to parse price string: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unexpected price type: %T", current)
	}
}
