package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	customquery "github.com/tellor-io/layer/daemons/custom_query"
	"github.com/tellor-io/layer/daemons/lib/prices"
	"github.com/tellor-io/layer/daemons/pricefeed/client/types"
	"github.com/tellor-io/layer/utils"
)

// MedianLogData represents the structure for logging median price source data
type MedianLogData struct {
	Pair       string        `json:"Pair"`
	SourceData []SourcePrice `json:"SourceData"`
}

// SourcePrice represents a price from a specific source
type SourcePrice struct {
	Exchange string  `json:"Exchange"`
	Price    float64 `json:"Price"`
}

func (c *Client) median(querydata []byte) (string, error) {
	querydatastr := hex.EncodeToString(querydata)

	for _, marketParam := range c.MarketParams {
		if strings.EqualFold(marketParam.QueryData, querydatastr) {
			mv := c.MarketToExchange.GetValidMedianPricesWithSourceData([]types.MarketParam{marketParam}, time.Now())
			medianData, found := mv[marketParam.Id]
			if !found {
				return "", fmt.Errorf("no median values found for query data: %s", querydatastr)
			}
			value, err := prices.EncodePrice(float64(medianData.MedianPrice), marketParam.Exponent)
			if err != nil {
				return "", fmt.Errorf("faild to encode price: %w", err)
			}

			// Build source data for logging
			sourceData := make([]SourcePrice, len(medianData.SourceData))
			for i, ep := range medianData.SourceData {
				// Convert uint64 price to float64 using the market's exponent
				priceFloat := float64(prices.PriceToFloat32ForLogging(ep.Price, marketParam.Exponent))
				sourceData[i] = SourcePrice{
					Exchange: ep.ExchangeId,
					Price:    priceFloat,
				}
			}

			// Create log data structure
			logData := MedianLogData{
				Pair:       marketParam.Pair,
				SourceData: sourceData,
			}

			// Marshal to JSON and log
			logJSON, err := json.Marshal(logData)
			if err != nil {
				c.logger.Error("Failed to marshal median log data", "error", err)
			} else {
				c.logger.Info("Median Price Source Data", "data", string(logJSON))
			}

			c.logger.Info("Median Value", "pair", marketParam.Pair, "price", float64(medianData.MedianPrice))
			return value, nil
		}
	}
	// if can't find it here then check custom query config
	queryId := utils.QueryIDFromData(querydata)
	queryIdStr := hex.EncodeToString(queryId)
	queryConfig, ok := c.Custom_query[queryIdStr]
	if !ok {
		return "", fmt.Errorf("no config found for query data: %s", querydatastr)
	}
	results, err := customquery.FetchPrice(context.Background(), queryConfig, c.MarketToExchange)
	if err != nil {
		return "", fmt.Errorf("failed to fetch price: %w", err)
	}

	// Log source data for custom query path
	if len(results.RawResults) > 0 {
		// Filter successful results
		successfulResults := make([]customquery.Result, 0)
		for _, result := range results.RawResults {
			if result.Err == nil {
				successfulResults = append(successfulResults, result)
			}
		}

		if len(successfulResults) > 0 {
			// Extract pair from first successful result's MarketId
			pair := successfulResults[0].MarketId
			if pair == "" {
				// Fallback: try to get from any result
				for _, result := range results.RawResults {
					if result.MarketId != "" {
						pair = result.MarketId
						break
					}
				}
			}

			// Build source data for logging
			sourceData := make([]SourcePrice, len(successfulResults))
			for i, result := range successfulResults {
				sourceData[i] = SourcePrice{
					Exchange: result.SourceId,
					Price:    result.Value,
				}
			}

			// Create log data structure
			logData := MedianLogData{
				Pair:       pair,
				SourceData: sourceData,
			}

			// Marshal to JSON and log
			logJSON, err := json.Marshal(logData)
			if err != nil {
				c.logger.Error("Failed to marshal median log data", "error", err)
			} else {
				c.logger.Info("Median Price Source Data", "data", string(logJSON))
			}
		}
	}

	return results.EncodedValue, nil
}
