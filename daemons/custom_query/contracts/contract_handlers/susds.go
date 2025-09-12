package contract_handlers

import (
	"context"
	"fmt"
	"math/big"

	reader "github.com/tellor-io/layer/daemons/custom_query/contracts/contract_reader"
	pricefeedservertypes "github.com/tellor-io/layer/daemons/server/types/pricefeed"
)

var _ ContractHandler = (*SUSDSHandler)(nil)

type SUSDSHandler struct{}

func (s *SUSDSHandler) FetchValue(
	ctx context.Context, reader *reader.Reader,
	priceCache *pricefeedservertypes.MarketToExchangePrices,
) (float64, error) {
	result, err := reader.ReadContract(ctx, SUSDS_CONTRACT, "convertToAssets(uint256) returns (uint256)", []string{"1000000000000000000"})
	if err != nil {
		return 0, err
	}

	valueInUsdWei := new(big.Int).SetBytes(result)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	divisorFloat := new(big.Float).SetInt(divisor)

	valueInUsdFloat := new(big.Float).SetInt(valueInUsdWei)
	usdValue := new(big.Float).Quo(valueInUsdFloat, divisorFloat)

	value, _ := usdValue.Float64()

	fmt.Println("sUSDS Price (USD):", value)

	return value, nil
}
