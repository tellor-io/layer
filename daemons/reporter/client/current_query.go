package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tellor-io/layer/utils"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (c *Client) CurrentQuery(ctx context.Context) (string, error) {
	response, err := c.OracleQueryClient.CurrentCyclelistQuery(ctx, &oracletypes.QueryCurrentCyclelistQueryRequest{})
	if err != nil {
		return "", fmt.Errorf("error calling 'CurrentCyclelistQuery': %v", err)
	}
	qid, err := utils.QueryIDFromDataString(response.Querydata)
	if err != nil {
		return "", fmt.Errorf("error getting query id from data string: %v", err)
	}

	c.logger.Info("ReporterDaemon", "next query id in cycle list", hex.EncodeToString(qid))
	return strings.ToLower(response.Querydata), nil
}
