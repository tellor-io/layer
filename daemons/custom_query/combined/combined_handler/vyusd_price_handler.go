package combined_handler

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	daemonconstants "github.com/tellor-io/layer/daemons/constants"
	contractreader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	rpcreader "github.com/tellor-io/layer/daemons/custom_query/rpc/rpc_reader"
	exchange_common "github.com/tellor-io/layer/daemons/exchange_common"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

const VYUSD_CONTRACT = "0x2e3c5e514eef46727de1fe44618027a9b70d92fc"

// VYUSDPriceHandler calculates vYUSD price by dividing the contract conversion rate by the USDC spot price
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

	fetcher := NewParallelFetcher()

	fetcher.FetchContract(
		ctx,
		"conversion_rate",
		contractReader,
		VYUSD_CONTRACT,
		"exchangeRateScaled() returns (uint256)",
		nil,
	)

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

	// Get USDC/USD price from in-memory cache (median across exchanges)
	if priceCache == nil {
		return 0, fmt.Errorf("price cache is not available")
	}

	usdcParam, found := daemonconstants.StaticMarketParamsConfig[exchange_common.USDCUSD_ID]
	if !found {
		return 0, fmt.Errorf("no valid USDC-USD market param found")
	}
	usdcPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdcParam}, time.Now())
	usdcRaw, ok := usdcPriceMap[exchange_common.USDCUSD_ID]
	if !ok {
		return 0, fmt.Errorf("no valid USDC-USD price found in cache")
	}
	usdcPrice := float64(usdcRaw) * math.Pow10(int(usdcParam.Exponent))
	if usdcPrice == 0 {
		return 0, fmt.Errorf("invalid USDC-USD price (zero)")
	}

	// Divide exchange rate by USDC price
	result := conversionRateFloat / usdcPrice

	fmt.Printf("vYUSD Price: exchange_rate=%f, usdc_price=%f, final=%f\n", conversionRateFloat, usdcPrice, result)

	return result, nil
}
