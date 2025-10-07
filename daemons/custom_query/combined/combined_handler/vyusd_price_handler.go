package combined_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	gometrics "github.com/hashicorp/go-metrics"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	"github.com/tellor-io/layer/daemons/lib/metrics"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

const VYUSD_CONTRACT = "0x2e3c5e514eef46727de1fe44618027a9b70d92fc"

// VYUSDPriceHandler calculates vYUSD price by multiplying the contract conversion rate with an RPC price
type VYUSDPriceHandler struct{}

func init() {
	RegisterHandler("vyusd_price", &VYUSDPriceHandler{})
}

func (h *VYUSDPriceHandler) FetchValue(
	ctx context.Context,
	contractReaders map[string]*contractreader.Reader,
	rpcReaders map[string]*rpcreader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
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
		VYUSD_CONTRACT,
		"exchangeRateScaled() returns (uint256)",
		nil,
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
		priceBytes, err := fetcher.GetRPCBytes("price_" + name)
		if err != nil {
			fmt.Printf("Warning: failed to get price from %s: %v\n", name, err)
			continue
		}

		price, err := h.extractPrice(priceBytes, reader.ResponsePath)
		if err != nil {
			fmt.Printf("Warning: failed to extract price from %s: %v\n", name, err)
			continue
		}

		telemetry.SetGaugeWithLabels(
			[]string{metrics.PricefeedDaemon, metrics.PriceEncoderUpdatePrice},
			float32(price),
			[]gometrics.Label{
				metrics.GetLabelForStringValue(metrics.MarketId, "VYUSD-USD"),
				metrics.GetLabelForStringValue(metrics.ExchangeId, name),
			},
		)

		prices = append(prices, price)
	}

	if len(prices) == 0 {
		return 0, fmt.Errorf("failed to get any prices from RPC endpoints")
	}

	// Calculate median price
	medianPrice := h.calculateMedian(prices)

	// Multiply conversion rate by median price
	result := conversionRateFloat * medianPrice

	fmt.Printf("vYUSD Price: conversion_rate=%f, median_price=%f, final=%f\n", conversionRateFloat, medianPrice, result)

	return result, nil
}

func (h *VYUSDPriceHandler) calculateMedian(prices []float64) float64 {
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

func (h *VYUSDPriceHandler) extractPrice(data []byte, path []string) (float64, error) {
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
