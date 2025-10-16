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

var _ ContractHandler = (*YieldFiYeth)(nil)

type YieldFiYeth struct{}

func (r *YieldFiYeth) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, YIELDFI_YETH_CONTRACT, "exchangeRate() returns (uint256)", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call exchangeRate: %w", err)
	}

	// Get yieldFi-yeth exchange rate from contract (how much ETH 1 yieldFi-yeth is worth)
	yieldFiExchangeRate := new(big.Int).SetBytes(result)
	ethUsdMarketParam, found := constants.StaticMarketParamsConfig[exchange_common.ETHUSD_ID]
	if !found {
		return 0, errors.New("no valid ETH-USD market param found")
	}
	// Get ETH/USD price from cache
	ethusdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*ethUsdMarketParam}, time.Now())
	ethusdPrice, found := ethusdPriceMap[exchange_common.ETHUSD_ID]
	if !found {
		return 0, errors.New("no valid ETH-USD price found")
	}

	// Multiply: rETH exchange rate (18 decimals) * ETH price (with market param decimals)
	ethPriceBig := new(big.Int).SetUint64(ethusdPrice)
	value := new(big.Int).Mul(yieldFiExchangeRate, ethPriceBig)

	// Calculate total decimals to remove:
	// - 18 from yieldFi exchange rate
	// - Plus the market param decimals (negative exponent means positive decimals)
	totalDecimals := 18 + (-ethUsdMarketParam.Exponent)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(totalDecimals)), nil)

	// Convert to big.Float for final division to get USD value
	valueFloat := new(big.Float).SetInt(value)
	divisorFloat := new(big.Float).SetInt(divisor)
	finalResult := new(big.Float).Quo(valueFloat, divisorFloat)

	finalValue, _ := finalResult.Float64()
	fmt.Printf("YieldFi-yETH Price (USD): $%.2f\n", finalValue)

	return finalValue, nil
}
