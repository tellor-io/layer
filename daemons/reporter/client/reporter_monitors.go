package client

import (
	"bytes"
	"context"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func (c *Client) MonitorCyclelistQuery(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	prevQueryData := []byte{}

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

		err = c.WaitForBlockHeight(ctx, int64(querymeta.Expiration))
		if err != nil {
			c.logger.Error("Error waiting for block height", "error", err)
		}
	}
}

func (c *Client) MonitorTokenBridgeReports(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	var localWG sync.WaitGroup
	for {
		localWG.Add(1)
		go func() {
			defer localWG.Done()
			err := c.generateDepositmessages(context.Background())
			if err != nil {
				c.logger.Error("Error generating deposit messages: ", err)
			}
		}()
		localWG.Wait()

		time.Sleep(4 * time.Minute)
	}
}
