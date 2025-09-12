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

var _ ContractHandler = (*RocketPoolETHHandler)(nil)

type RocketPoolETHHandler struct{}

func (r *RocketPoolETHHandler) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, RETH_CONTRACT, "getExchangeRate() returns (uint256)", nil)
	if err != nil {
		return 0, err
	}
	// Get rETH exchange rate from contract (how much ETH 1 rETH is worth)
	rethExchangeRate := new(big.Int).SetBytes(result)
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

	ethPriceBig := new(big.Int).SetUint64(ethusdPrice)

	value := new(big.Int).Mul(rethExchangeRate, ethPriceBig)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	value.Div(value, divisor)
	fmt.Println("Rocket Pool ETH Price (USD):", value.String())

	valueFloat, _ := value.Float64()
	return valueFloat, nil
}
