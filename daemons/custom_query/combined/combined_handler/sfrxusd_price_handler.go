package combined_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	log "github.com/sirupsen/logrus"
	contract_handlers "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_handlers"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

const (
	MIN_FRX_SOURCES    = 2
	MAX_FRX_SPREAD_PCT = 50.0 // Maximum allowed spread between FRX/USD prices
)

// SFRXUSDPriceHandler calculates sFRXUSD price by multiplying the fundamental rate by FRX/USD spot price
// Note: Uses RPC sources (CoinGecko, CoinPaprika, Curve) because FRX is not available on standard CEX exchanges
type SFRXUSDPriceHandler struct{}

func init() {
	RegisterHandler("sfrxusd_price", &SFRXUSDPriceHandler{})
}

func (h *SFRXUSDPriceHandler) FetchValue(
	ctx context.Context,
	contractReaders map[string]*contractreader.Reader,
	rpcReaders map[string]*rpcreader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	// validate eth contract reader
	contractReader, exists := contractReaders["ethereum"]
	if !exists {
		return 0, fmt.Errorf("ethereum contract reader not found")
	}

	fetcher := NewParallelFetcher()

	// get sFrx contract data
	fetcher.FetchContract(
		ctx,
		"total_assets",
		contractReader,
		contract_handlers.SFRXUSD_CONTRACT,
		"totalAssets() returns (uint256)",
		nil,
	)

	fetcher.FetchContract(
		ctx,
		"total_supply",
		contractReader,
		contract_handlers.SFRXUSD_CONTRACT,
		"totalSupply() returns (uint256)",
		nil,
	)

	// get frx/usd spot price
	if reader, exists := rpcReaders["coingecko"]; exists {
		fetcher.FetchRPC(ctx, "frx_coingecko", reader)
	} else {
		log.Warn("[sFRXUSD] CoinGecko reader not available")
	}
	if reader, exists := rpcReaders["curve"]; exists {
		fetcher.FetchRPC(ctx, "frx_curve", reader)
	} else {
		log.Warn("[sFRXUSD] Curve reader not available")
	}
	if reader, exists := rpcReaders["coinpaprika"]; exists {
		fetcher.FetchRPC(ctx, "frx_coinpaprika", reader)
	} else {
		log.Warn("[sFRXUSD] CoinPaprika reader not available")
	}

	fetcher.Wait()

	// parse contract data
	totalAssetsBytes, err := fetcher.GetBytes("total_assets")
	if err != nil {
		return 0, fmt.Errorf("failed to get totalAssets: %w", err)
	}

	totalSupplyBytes, err := fetcher.GetBytes("total_supply")
	if err != nil {
		return 0, fmt.Errorf("failed to get totalSupply: %w", err)
	}

	totalAssets := new(big.Int).SetBytes(totalAssetsBytes)
	totalSupply := new(big.Int).SetBytes(totalSupplyBytes)

	// prevent division by zero
	if totalSupply.Sign() == 0 {
		return 0, fmt.Errorf("invalid total supply: zero")
	}

	// calculate fundamental rate (total assets / total supply)
	fundamentalRate := new(big.Float).Quo(
		new(big.Float).SetInt(totalAssets),
		new(big.Float).SetInt(totalSupply),
	)
	fundamentalRateFloat, _ := fundamentalRate.Float64()
	log.Infof("[sFRXUSD] Fundamental rate: %f", fundamentalRateFloat)

	var frxPrices []float64

	// Parse CoinGecko response
	if result, err := fetcher.GetBytes("frx_coingecko"); err == nil {
		var cgResponse map[string]map[string]float64
		if err := json.Unmarshal(result, &cgResponse); err == nil {
			if fraxData, exists := cgResponse["frax"]; exists {
				if price, ok := fraxData["usd"]; ok {
					frxPrices = append(frxPrices, price)
				} else {
					log.Warn("[sFRXUSD] CoinGecko response missing frax.usd field")
				}
			} else {
				log.Warn("[sFRXUSD] CoinGecko response missing frax key")
			}
		} else {
			log.Warnf("[sFRXUSD] Failed to parse CoinGecko JSON: %v", err)
		}
	} else {
		log.Warnf("[sFRXUSD] Failed to fetch CoinGecko data: %v", err)
	}

	// Parse Curve response
	if result, err := fetcher.GetBytes("frx_curve"); err == nil {
		var curveResponse struct {
			Data struct {
				UsdPrice float64 `json:"usd_price"`
			} `json:"data"`
		}
		if err := json.Unmarshal(result, &curveResponse); err == nil {
			if curveResponse.Data.UsdPrice > 0 {
				frxPrices = append(frxPrices, curveResponse.Data.UsdPrice)
			} else {
				log.Warn("[sFRXUSD] Curve response has zero or missing price")
			}
		} else {
			log.Warnf("[sFRXUSD] Failed to parse Curve JSON: %v", err)
		}
	} else {
		log.Warnf("[sFRXUSD] Failed to fetch Curve data: %v", err)
	}

	// Parse CoinPaprika response
	if result, err := fetcher.GetBytes("frx_coinpaprika"); err == nil {
		var cpResponse struct {
			Quotes struct {
				USD struct {
					Price float64 `json:"price"`
				} `json:"USD"`
			} `json:"quotes"`
		}
		if err := json.Unmarshal(result, &cpResponse); err == nil {
			if cpResponse.Quotes.USD.Price > 0 {
				frxPrices = append(frxPrices, cpResponse.Quotes.USD.Price)
			} else {
				log.Warn("[sFRXUSD] CoinPaprika response has zero or missing price")
			}
		} else {
			log.Warnf("[sFRXUSD] Failed to parse CoinPaprika JSON: %v", err)
		}
	} else {
		log.Warnf("[sFRXUSD] Failed to fetch CoinPaprika data: %v", err)
	}

	if len(frxPrices) < MIN_FRX_SOURCES {
		return 0, fmt.Errorf("insufficient FRX/USD prices: got %d, need at least %d", len(frxPrices), MIN_FRX_SOURCES)
	}

	// calculate median frx/usd spot price
	sort.Float64s(frxPrices)

	// validate spread between min and max prices
	minPrice := frxPrices[0]
	maxPrice := frxPrices[len(frxPrices)-1]
	spreadPercent := ((maxPrice - minPrice) / minPrice) * 100

	if spreadPercent > MAX_FRX_SPREAD_PCT {
		log.Warnf("[sFRXUSD] FRX/USD prices show excessive spread: %.2f%% (max: %.2f%%), prices: %v",
			spreadPercent, MAX_FRX_SPREAD_PCT, frxPrices)
		return 0, fmt.Errorf("FRX/USD price spread of %.2f%% exceeds maximum allowed %.2f%%",
			spreadPercent, MAX_FRX_SPREAD_PCT)
	}

	log.Infof("[sFRXUSD] FRX/USD price spread: %.2f%%, prices: %v", spreadPercent, frxPrices)

	n := len(frxPrices)
	var frxUsdPrice float64
	if n%2 == 0 {
		frxUsdPrice = (frxPrices[n/2-1] + frxPrices[n/2]) / 2
	} else {
		frxUsdPrice = frxPrices[n/2]
	}

	if frxUsdPrice <= 0 {
		return 0, fmt.Errorf("invalid median FRX/USD price: %.6f", frxUsdPrice)
	}

	log.Infof("[sFRXUSD] Median FRX/USD price: $%.6f", frxUsdPrice)

	// final result = fundamental rate * frx/usd spot price
	result := fundamentalRateFloat * frxUsdPrice

	return result, nil
}
