package client

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/tellor-io/layer/daemons/lib/prices"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
)

func (c *Client) median(querydata []byte) (string, error) {
	querydatastr := hex.EncodeToString(querydata)

	for _, marketParam := range c.MarketParams {
		if strings.EqualFold(marketParam.QueryData, querydatastr) {
			mv := c.MarketToExchange.GetValidMedianPrices([]types.MarketParam{marketParam}, time.Now())
			val, found := mv[marketParam.Id]
			if !found {
				return "", fmt.Errorf("no median values found for query data: %s", querydatastr)
			}
			value, err := prices.EncodePrice(float64(val), marketParam.Exponent)
			if err != nil {
				return "", fmt.Errorf("faild to encode price: %w", err)
			}
			c.logger.Info("Median Value", "pair", marketParam.Pair, "price", float64(val))
			return value, nil
		}
	}
	return "", fmt.Errorf("no market param found for query data: %s", querydatastr)
}
