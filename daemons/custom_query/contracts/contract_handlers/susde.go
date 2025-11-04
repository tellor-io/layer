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

var _ ContractHandler = (*SUSDEUSD)(nil)

type SUSDEUSD struct{}

// FetchValue calculates SUSDE price in USD by:
// 1. Getting SUSDE/USD exchange rate from contract (18 decimals)
// 2. Getting USDE/USD price from cache
// 3. Multiplying: (SUSDE/USDE rate) * (USDE/USD price) = SUSDE/USD price
// 4. Adjusting for decimals: 18 from exchange rate + market param decimals
func (r *SUSDEUSD) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, SUSDE_CONTRACT, "convertToAssets(uint256) returns (uint256)", []string{"1000000000000000000"})
	if err != nil {
		return 0, fmt.Errorf("failed to call convertToAssets: %w", err)
	}

	// Get SUSDE exchange rate from contract (how much USDE 1 SUSDE is worth)
	susdeExchangeRate := new(big.Int).SetBytes(result)
	usdeUsdMarketParam, found := constants.StaticMarketParamsConfig[exchange_common.USDEUSD_ID]
	if !found {
		return 0, errors.New("no valid USDE-USD market param found")
	}
	// Get USDE/USD price from cache
	usdeusdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*usdeUsdMarketParam}, time.Now())
	usdeusdPrice, found := usdeusdPriceMap[exchange_common.USDEUSD_ID]
	if !found {
		return 0, errors.New("no valid USDE-USD price found")
	}

	usdePriceBig := new(big.Int).SetUint64(usdeusdPrice)
	value := new(big.Int).Mul(susdeExchangeRate, usdePriceBig)
	// Calculate total decimals to remove:
	// - 18 from SUSDE exchange rate
	// - Plus the market param decimals (negative exponent means positive decimals)
	totalDecimals := 18 + (-usdeUsdMarketParam.Exponent)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(totalDecimals)), nil)

	// Convert to big.Float for final division to get USD value
	valueFloat := new(big.Float).SetInt(value)
	divisorFloat := new(big.Float).SetInt(divisor)
	finalResult := new(big.Float).Quo(valueFloat, divisorFloat)

	finalValue, _ := finalResult.Float64()
	fmt.Printf("SUSDE Price (USD): $%.2f\n", finalValue)

	return finalValue, nil
}
