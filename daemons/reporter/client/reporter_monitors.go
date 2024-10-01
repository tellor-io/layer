package client

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (c *Client) MonitorCyclelistQuery(ctx context.Context) {
	prevQueryData := []byte{}
	queryRes, err := c.OracleQueryClient.Params(ctx, &oracletypes.QueryParamsRequest{})
	if err != nil {
		c.logger.Error(fmt.Sprintf("ERROR get offset param: %v", err))
		return
	}
	offset := queryRes.Params.Offset
	for {
		querydata, querymeta, err := c.CurrentQuery(ctx)
		if err != nil {
			// log error
			c.logger.Error("getting current query", "error", err)
		}
		if bytes.Equal(querydata, prevQueryData) || commitedIds[querymeta.Id] {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go func(ctx context.Context, qd []byte, qm *oracletypes.QueryMeta) {
			err := c.CyclelistMessages(ctx, querydata, qm)
			if err != nil {
				c.logger.Error("Generating CycleList message", "error", err)
			}
		}(ctx, querydata, querymeta)

		currentTime := time.Now()
		time.Sleep(querymeta.Expiration.Add(offset).Sub(currentTime))
	}
}

func (c *Client) MonitorTokenBridgeReports(ctx context.Context) {
	var wg sync.WaitGroup
	for {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := c.generateDepositmessages(context.Background())
			if err != nil {
				c.logger.Error("Error generating deposit messages: ", err)
			}
		}()

		time.Sleep(5 * time.Minute)
	}
}
