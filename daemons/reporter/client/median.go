package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/lib/prices"
)

func (c *Client) median(querydata string) (string, error) {
	params := c.MarketParams
	exchPrices := c.MarketToExchange
	mapping := exchPrices.GetValidMedianPrices(params, time.Now())
	fmt.Println("Price Mapping:", mapping)

	mapQueryDataToMarketParams := make(map[string]types.MarketParam)
	for _, marketParam := range c.MarketParams {
		mapQueryDataToMarketParams[strings.ToLower(marketParam.QueryData)] = marketParam
	}
	mp, found := mapQueryDataToMarketParams[querydata]
	if !found {
		return "", fmt.Errorf("no market param found for query data: %s", querydata)
	}
	mv := c.MarketToExchange.GetValidMedianPrices([]types.MarketParam{mp}, time.Now())
	val, found := mv[mp.Id]
	if !found {
		return "", fmt.Errorf("no median values found for query data: %s", querydata)
	}

	value, err := prices.EncodePrice(float64(val), mp.Exponent)
	if err != nil {
		return "", fmt.Errorf("faild to encode price: %v", err)
	}
	return value, nil
}
