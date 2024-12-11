package client

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/types/query"
)

func (c *Client) MonitorCyclelistQuery(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	prevQueryData := []byte{}

	for {
		querydata, querymeta, err := c.CurrentQuery(ctx)
		if err != nil {
			// log error
			c.logger.Error("getting current query", "error", err)
			continue
		}

		if querymeta == nil {
			c.logger.Error("QueryMeta is nil")
			continue
		}

		mutex.RLock()
		committed := commitedIds[querymeta.Id]
		mutex.RUnlock()
		if bytes.Equal(querydata, prevQueryData) || committed {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go func(ctx context.Context, qd []byte, qm *oracletypes.QueryMeta) {
			err := c.GenerateAndBroadcastSpotPriceReport(ctx, querydata, qm)
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

func (c *Client) MonitorForTippedQueries(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		res, err := c.OracleQueryClient.TippedQueries(ctx, &oracletypes.QueryTippedQueriesRequest{
			Pagination: &query.PageRequest{
				Offset: 0,
			},
		})
		if err != nil {
			c.logger.Error("Error querying for TippedQueries: ", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if len(res.Queries) == 0 {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		status, err := c.cosmosCtx.Client.Status(ctx)
		if err != nil {
			c.logger.Info("Error getting status from client: ", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		height := uint64(status.SyncInfo.LatestBlockHeight)

		// Create a new WaitGroup for this batch of tips
		var batchWG sync.WaitGroup

		for i := 0; i < len(res.Queries); i++ {
			mutex.RLock()
			committed := commitedIds[res.Queries[i].Id]
			mutex.RUnlock()
			if height > res.Queries[i].Expiration || committed || strings.EqualFold(res.Queries[i].QueryType, "SpotPrice") {
				continue
			}

			batchWG.Add(1)
			go func(query *oracletypes.QueryMeta) {
				defer batchWG.Done()
				err := c.GenerateAndBroadcastSpotPriceReport(ctx, query.GetQueryData(), query)
				if err != nil {
					c.logger.Error("Error generating report for tipped query: ", err)
				} else {
					c.logger.Info("Broadcasted report for tipped query")
				}
			}(res.Queries[i])
		}

		// Wait for all reports in this batch to complete
		batchWG.Wait()

		// Add a small delay between batches to prevent overwhelming the system
		time.Sleep(500 * time.Millisecond)
	}
}
