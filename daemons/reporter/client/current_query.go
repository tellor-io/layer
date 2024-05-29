package client

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (c *Client) CurrentQuery(ctx context.Context) ([]byte, error) {
	response, err := c.OracleQueryClient.CurrentCyclelistQuery(ctx, &oracletypes.QueryCurrentCyclelistQueryRequest{})
	if err != nil {
		return nil, fmt.Errorf("error calling 'CurrentCyclelistQuery': %v", err)
	}
	querydata, err := utils.QueryBytesFromString(response.QueryData)
	if err != nil {
		return nil, fmt.Errorf("error parsing query id from response: %v", err)
	}

	c.logger.Info("ReporterDaemon", "next query id in cycle list", hex.EncodeToString(querydata))
	return querydata, nil
}
