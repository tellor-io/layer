package contract_handlers

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	"github.com/tellor-io/layer/daemons/exchange_common"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

var _ ContractHandler = (*YieldFiYusd)(nil)

type YieldFiYusd struct{}

// FetchValue calculates yieldFi-yUSD price in USD by:
// 1. Getting yieldFi-yUSD/USDC exchange rate from contract (18 decimals)
// 2. Getting USDC/USD price from cache
// 3. Multiplying: (yieldFi-yUSD/USDC rate) * (USDC/USD price) = yieldFi-yUSD/USD price
// 4. Adjusting for decimals: 18 from exchange rate + market param decimals
func (r *YieldFiYusd) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, YIELDFI_YUSD_CONTRACT, "exchangeRateScaled() returns (uint256)", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call exchangeRateScaled: %w", err)
	}

	// Get yieldFi-yUSD exchange rate from contract (how much USD 1 yieldFi-yUSD is worth)
	yieldFiExchangeRate := new(big.Int).SetBytes(result)
	usdcUsdMarketParam, found := constants.StaticMarketParamsConfig[exchange_common.USDCUSD_ID]
	if !found {
		return 0, errors.New("no valid USDC-USD market param found")
	}
	// Get USDC/USD price from cache
	usdcusdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdcUsdMarketParam}, time.Now())
	usdcusdPrice, found := usdcusdPriceMap[exchange_common.USDCUSD_ID]
	if !found {
		return 0, errors.New("no valid USDC-USD price found")
	}

	usdcPriceBig := new(big.Int).SetUint64(usdcusdPrice)
	value := new(big.Int).Mul(yieldFiExchangeRate, usdcPriceBig)

	// Calculate total decimals to remove:
	// - 18 from yieldFi exchange rate
	// - Plus the market param decimals (negative exponent means positive decimals)
	totalDecimals := 18 + (-usdcUsdMarketParam.Exponent)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(totalDecimals)), nil)

	// Convert to big.Float for final division to get USD value
	valueFloat := new(big.Float).SetInt(value)
	divisorFloat := new(big.Float).SetInt(divisor)
	finalResult := new(big.Float).Quo(valueFloat, divisorFloat)

	finalValue, _ := finalResult.Float64()
	fmt.Printf("YieldFi-yUSD Price (USD): $%.2f\n", finalValue)

	return finalValue, nil
}
