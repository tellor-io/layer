package client

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/shirou/gopsutil/v3/process"
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

	// Recovery function for goroutines
	handlePanic := func() {
		if r := recover(); r != nil {
			c.logger.Error("Recovered from panic in token bridge monitor",
				"panic", r,
				"stack", string(debug.Stack()))
		}
		localWG.Done()
	}

	for {
		select {
		case <-ctx.Done():
			// Wait for any in-progress operations
			localWG.Wait()
			return
		default:
			localWG.Add(1)
			go func() {
				defer handlePanic()

				// Use timeout context for deposit generation
				genCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				err := c.GenerateDepositMessages(genCtx)
				if err != nil {
					c.logger.Error("Error generating deposit messages",
						"error", err)
				}
			}()

			// Wait for current operation to complete before starting next
			localWG.Wait()
			time.Sleep(10 * time.Second)
			c.LogProcessStats()
		}
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

func (c *Client) LogProcessStats() {
	count := runtime.NumGoroutine()
	c.logger.Info(fmt.Sprintf("Number of Goroutines: %d\n", count))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.logger.Info(fmt.Sprintf("Memory Stats: { 'alloc': %d, 'total alloc': %d, 'mallocs': %d, 'frees': %d, 'heap released': %d}", m.Alloc, m.TotalAlloc, m.Mallocs, m.Frees, m.HeapReleased))

	pid := int32(os.Getpid())
	p, err := process.NewProcess(pid)
	if err != nil {
		c.logger.Error(fmt.Sprintf("Error getting process info: %v\n", err))
		return
	}

	// Get CPU usage percentage
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		c.logger.Error(fmt.Sprintf("Error getting CPU percent: %v\n", err))
		return
	}

	numThreads, err := p.NumThreads()
	if err != nil {
		c.logger.Error(fmt.Sprintf("Error getting num of threads: %v\n", numThreads))
		return
	}

	c.logger.Info(fmt.Sprintf("CPU Usage: %.2f%%, Num of threads: %d\n", cpuPercent, numThreads))
}
