package contract_handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/tellor-io/layer/daemons/constants"
	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	"github.com/tellor-io/layer/daemons/exchange_common"
	marketParam "github.com/tellor-io/layer/daemons/pricefeed/client/types"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

var _ ContractHandler = (*WSTETHHandler)(nil)

type WSTETHHandler struct{}

func (s *WSTETHHandler) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, WSTETH_CONTRACT, "getStETHByWstETH(uint256) returns (uint256)", []string{"1000000000000000000"})
	if err != nil {
		return 0, err
	}

	stEthPerWstEth := new(big.Int).SetBytes(result)

	stEthUsdMarketParam, found := constants.StaticMarketParamsConfig[exchange_common.STETHUSD_ID]
	if !found {
		return 0, errors.New("no valid stETH-USD market param found")
	}
	// Get stETH/USD price from cache
	stEthUsdPriceMap := priceCache.GetValidMedianPrices([]marketParam.MarketParam{*stEthUsdMarketParam}, time.Now())
	stEthusdPrice, found := stEthUsdPriceMap[exchange_common.STETHUSD_ID]
	if !found {
		return 0, errors.New("no valid stETH-USD price found")
	}
	stEthPriceBig := new(big.Int).SetUint64(stEthusdPrice)

	value := new(big.Int).Mul(stEthPerWstEth, stEthPriceBig)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	value.Div(value, divisor)

	scaleFactor := math.Pow(10, float64(stEthUsdMarketParam.Exponent)) // exponent is negative
	valueFloat := float64(value.Uint64()) * scaleFactor

	fmt.Printf("wstETH Price: $%.2f USD\n", valueFloat)

	return valueFloat, nil

}
