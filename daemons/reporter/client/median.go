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

func (c *Client) median(querydata []byte) (encodedValue string, rawPrice float64, err error) {
	querydatastr := hex.EncodeToString(querydata)

	for _, marketParam := range c.MarketParams {
		if strings.EqualFold(marketParam.QueryData, querydatastr) {
			mv := c.MarketToExchange.GetValidMedianPrices([]types.MarketParam{marketParam}, time.Now())
			val, found := mv[marketParam.Id]
			if !found {
				return "", 0, fmt.Errorf("no median values found for query data: %s", querydatastr)
			}
			value, err := prices.EncodePrice(float64(val), marketParam.Exponent)
			if err != nil {
				return "", 0, fmt.Errorf("faild to encode price: %w", err)
			}
			c.logger.Info("Median Value", "pair", marketParam.Pair, "price", float64(val))
			return value, float64(val), nil
		}
	}
	// if can't find it here then check custom query config
	queryId := utils.QueryIDFromData(querydata)
	queryIdStr := hex.EncodeToString(queryId)
	queryConfig, ok := c.Custom_query[queryIdStr]
	if !ok {
		return "", 0, fmt.Errorf("no config found for query data: %s", querydatastr)
	}
	results, err := customquery.FetchPrice(context.Background(), queryConfig, c.MarketToExchange)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch price: %w", err)
	}
	// For custom queries, we return 0 for raw price (price guard won't apply)
	return results.EncodedValue, 0, nil
}
