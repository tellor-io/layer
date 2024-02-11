package coingecko

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/tellor-io/layer/daemons/exchange_common"
	price_function "github.com/tellor-io/layer/daemons/pricefeed/client/sources"
	"github.com/tellor-io/layer/daemons/pricefeed/types"
	"github.com/tellor-io/layer/lib"
)

func CoingeckoPriceFunction(
	response *http.Response,
	tickerToExponent map[string]int32,
	resolver types.Resolver,
) (tickerToPrice map[string]uint64, unavailableTickers map[string]error, err error) {
	// Get ticker. The API response should only contain information for one market.
	ticker, _, err := price_function.GetOnlyTickerAndExponent(
		tickerToExponent,
		exchange_common.EXCHANGE_ID_COINGECKO,
	)
	if err != nil {
		return nil, nil, err
	}
	// Unmarshal response body.
	var coingeckoSimple map[string]map[string]float64
	err = json.NewDecoder(response.Body).Decode(&coingeckoSimple)
	if err != nil {
		return nil, nil, err
	}
	parsedTicker := strings.Split(ticker, "-")
	// Get the price for the ticker.
	price, ok := coingeckoSimple[parsedTicker[0]]
	if !ok {
		return nil, map[string]error{ticker: fmt.Errorf("ticker not found")}, nil
	}
	unsignedExponent := lib.AbsInt32(tickerToExponent[ticker])
	pow10 := new(big.Float).SetInt(lib.BigPow10(uint64(unsignedExponent)))
	// Convert the price to big float.
	priceBigFloat := new(big.Float).SetFloat64(price[parsedTicker[1]])
	value := reverseShiftFloatWithPow10(priceBigFloat, pow10, tickerToExponent[ticker])
	uint64Value, err := lib.ConvertBigFloatToUint64(value)
	if err != nil {
		return nil, nil, err
	}

	return map[string]uint64{ticker: uint64Value}, nil, nil
}
func reverseShiftFloatWithPow10(value *big.Float, pow10 *big.Float, exponent int32) *big.Float {
	if exponent == 0 {
		return value
	} else if exponent > 0 {
		return new(big.Float).Quo(value, pow10)
	} else { // exponent < 0
		return new(big.Float).Mul(value, pow10)
	}
}
