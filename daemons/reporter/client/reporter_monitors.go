package client

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"github.com/cosmos/cosmos-sdk/types/query"
)

const (
	defaultQueryTimeout = 10 * time.Second
	defaultTxTimeout    = 10 * time.Second
	defaultRetryDelay   = 200 * time.Millisecond
)

func (c *Client) MonitorCyclelistQuery(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	prevQueryData := []byte{}
	ticker := time.NewTicker(defaultRetryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
			querydata, querymeta, err := c.CurrentQuery(queryCtx)
			cancel()

			if err != nil || querymeta == nil {
				c.logger.Error("query failed", "error", err)
				continue
			}

			if bytes.Equal(querydata, prevQueryData) || commitedIds[querymeta.Id] {
				continue
			}

			// Handle report generation with timeout
			txCtx, cancel := context.WithTimeout(ctx, defaultTxTimeout)
			done := make(chan struct{})

			c.logger.Info(fmt.Sprintf("starting to generate spot price report at %d", time.Now().Unix()))
			go func() {
				defer close(done)
				err := c.GenerateAndBroadcastSpotPriceReport(txCtx, querydata, querymeta)
				if err != nil {
					c.logger.Error("report generation failed", "error", err)
				}
			}()

			select {
			case <-done:
				cancel()
			case <-txCtx.Done():
				c.logger.Error(fmt.Sprintf("report generation timed out at %d", time.Now().Unix()))
				cancel()
			}

			prevQueryData = querydata
		}
	}
}

func (c *Client) MonitorTokenBridgeReports(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			txCtx, cancel := context.WithTimeout(ctx, defaultTxTimeout)
			done := make(chan struct{})

			go func() {
				defer close(done)
				err := c.GenerateDepositMessages(txCtx)
				if err != nil {
					c.logger.Error("deposit generation failed", "error", err)
				}
			}()

			select {
			case <-done:
				cancel()
			case <-txCtx.Done():
				c.logger.Error("deposit generation timed out")
				cancel()
			}

			c.LogProcessStats()
		}
	}
}

func (c *Client) MonitorForTippedQueries(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(defaultRetryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
			res, err := c.OracleQueryClient.TippedQueries(queryCtx, &oracletypes.QueryTippedQueriesRequest{
				Pagination: &query.PageRequest{
					Offset: 0,
				},
			})
			cancel()

			if err != nil || len(res.Queries) == 0 {
				continue
			}

			status, err := c.cosmosCtx.Client.Status(ctx)
			if err != nil {
				continue
			}

			height := uint64(status.SyncInfo.LatestBlockHeight)

			for _, query := range res.Queries {
				if height > query.Expiration || commitedIds[query.Id] ||
					strings.EqualFold(query.QueryType, "SpotPrice") {
					continue
				}

				txCtx, cancel := context.WithTimeout(ctx, defaultTxTimeout)
				done := make(chan struct{})

				go func(q *oracletypes.QueryMeta) {
					defer close(done)
					err := c.GenerateAndBroadcastSpotPriceReport(txCtx, q.GetQueryData(), q)
					if err != nil {
						c.logger.Error("tipped query report failed", "error", err)
					}
				}(query)

				select {
				case <-done:
					cancel()
				case <-txCtx.Done():
					c.logger.Error("tipped query report timed out")
					cancel()
				}
			}
		}
	}
}

func (c *Client) WithdrawAndStakeEarnedRewardsPeriodically(ctx context.Context) {
	freqVar := os.Getenv("WITHDRAW_FREQUENCY")
	if freqVar == "" {
		freqVar = "43200" // default to being 12 hours or 43200 seconds
	}
	frequency, err := strconv.Atoi(freqVar)
	if err != nil {
		c.logger.Error("Could not start auto rewards withdrawal process due to incorrect parameter. Please enter the number of seconds to wait in between claiming rewards")
		return
	}

	for {
		valAddr := os.Getenv("REPORTERS_VALIDATOR_ADDRESS")
		if valAddr == "" {
			fmt.Println("Returning from Withdraw Monitor due to no validator address env variable was found")
			time.Sleep(time.Duration(frequency) * time.Second)
			continue
		}

		withdrawMsg := &reportertypes.MsgWithdrawTipLegacy{
			SelectorAddress:  c.accAddr.String(),
			ValidatorAddress: valAddr,
		}
		c.txChan <- TxChannelInfo{Msg: withdrawMsg, isBridge: false, NumRetries: 0}

		time.Sleep(time.Duration(frequency) * time.Second)
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
