package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	customquery "github.com/tellor-io/layer/daemons/custom_query"
	"github.com/tellor-io/layer/daemons/lib/prices"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/utils"
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
				return "", fmt.Errorf("failed to encode price: %w", err)
			}
			c.logger.Info("Median Value", "pair", marketParam.Pair, "price", float64(val))
			return value, nil
		}
	}
	// if can't find it here then check custom query config
	queryId := utils.QueryIDFromData(querydata)
	queryIdStr := hex.EncodeToString(queryId)
	queryConfig, ok := c.Custom_query[queryIdStr]
	if !ok {
		ethMarketParam := c.getEthUsdMarketParam()
		if ethMarketParam.Id == 0 {
			return "", fmt.Errorf("no config found for query data: %s", querydatastr)
		}
		mv := c.MarketToExchange.GetValidMedianPrices([]types.MarketParam{ethMarketParam}, time.Now())
		val, found := mv[ethMarketParam.Id]
		if !found {
			return "", fmt.Errorf("no median values found for query data: %s", querydatastr)
		}
		value, err := prices.EncodePrice(float64(val), ethMarketParam.Exponent)
		if err != nil {
			return "", fmt.Errorf("failed to encode price: %w", err)
		}
		c.logger.Info("Median Value", "pair", ethMarketParam.Pair, "price", float64(val))
		return value, nil
	}
	results, err := customquery.FetchPrice(context.Background(), queryConfig)
	if err != nil {
		return "", fmt.Errorf("failed to fetch price: %w", err)
	}
	return results.EncodedValue, nil
}

func (c *Client) getEthUsdMarketParam() types.MarketParam {
	for _, marketParam := range c.MarketParams {
		if strings.EqualFold(marketParam.QueryData, "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000") {
			return marketParam
		}
	}
	return types.MarketParam{}
}
