package client

import (
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"cosmossdk.io/log"

	"github.com/tellor-io/layer/utils"
)

type PriceGuard struct {
	lastPrices      map[string]float64   // queryID hex -> last submitted price
	lastUpdateTime  map[string]time.Time // queryID hex -> last update timestamp
	mu              sync.RWMutex
	globalThreshold float64            // default threshold for all queries
	queryThresholds map[string]float64 // per-queryID overrides
	maxAge          time.Duration      // max age before treating price as expired
	enabled         bool
	logger          log.Logger
}

func NewPriceGuard(globalThreshold float64, maxAge time.Duration, enabled bool, logger log.Logger) *PriceGuard {
	return &PriceGuard{
		lastPrices:      make(map[string]float64),
		lastUpdateTime:  make(map[string]time.Time),
		mu:              sync.RWMutex{},
		globalThreshold: globalThreshold,
		queryThresholds: make(map[string]float64),
		maxAge:          maxAge,
		enabled:         enabled,
		logger:          logger.With("component", "price_guard"),
	}
}

// SetQueryThreshold sets a specific threshold for a query ID (optional, for per-query overrides)
func (pg *PriceGuard) SetQueryThreshold(queryIdHex string, threshold float64) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.queryThresholds[queryIdHex] = threshold
}

// ShouldSubmit checks if the new price is within acceptable threshold
// Returns (shouldSubmit, reason)
func (pg *PriceGuard) ShouldSubmit(queryData []byte, newPrice float64) (bool, string) {
	if !pg.enabled {
		return true, ""
	}

	queryId := utils.QueryIDFromData(queryData)
	queryIdHex := hex.EncodeToString(queryId)

	pg.mu.RLock()
	lastPrice, exists := pg.lastPrices[queryIdHex]
	lastTime, timeExists := pg.lastUpdateTime[queryIdHex]
	threshold := pg.globalThreshold
	if customThreshold, ok := pg.queryThresholds[queryIdHex]; ok {
		threshold = customThreshold
	}
	pg.mu.RUnlock()

	// First submission for this query - allow it
	if !exists {
		pg.logger.Info("First submission for query, allowing",
			"queryId", queryIdHex,
			"price", newPrice,
		)
		return true, ""
	}

	// Check if last price is too old (expired)
	if timeExists && pg.maxAge > 0 {
		elapsed := time.Since(lastTime)
		if elapsed > pg.maxAge {
			pg.logger.Info("Last price expired, treating as new baseline",
				"queryId", queryIdHex,
				"elapsed", elapsed.String(),
				"maxAge", pg.maxAge.String(),
				"oldPrice", lastPrice,
				"newPrice", newPrice,
			)
			return true, ""
		}
	}

	// Prevent division by zero
	if lastPrice == 0 {
		pg.logger.Warn("Last price is zero, allowing submission",
			"queryId", queryIdHex,
			"newPrice", newPrice,
		)
		return true, ""
	}

	// Calculate percentage change
	change := math.Abs(newPrice-lastPrice) / lastPrice

	if change > threshold {
		reason := fmt.Sprintf(
			"Price change %.2f%% exceeds threshold %.2f%% (last: %.6f, new: %.6f)",
			change*100, threshold*100, lastPrice, newPrice,
		)
		pg.logger.Warn("Blocked submission due to price change",
			"queryId", queryIdHex,
			"changePercent", fmt.Sprintf("%.2f%%", change*100),
			"threshold", fmt.Sprintf("%.2f%%", threshold*100),
			"lastPrice", lastPrice,
			"newPrice", newPrice,
		)
		return false, reason
	}

	pg.logger.Debug("Price change within threshold",
		"queryId", queryIdHex,
		"changePercent", fmt.Sprintf("%.2f%%", change*100),
		"threshold", fmt.Sprintf("%.2f%%", threshold*100),
	)

	return true, ""
}

// UpdateLastPrice updates the cache after submission decision
func (pg *PriceGuard) UpdateLastPrice(queryData []byte, price float64) {
	if !pg.enabled {
		return
	}

	queryId := utils.QueryIDFromData(queryData)
	queryIdHex := hex.EncodeToString(queryId)

	pg.mu.Lock()
	pg.lastPrices[queryIdHex] = price
	pg.lastUpdateTime[queryIdHex] = time.Now()
	pg.mu.Unlock()

	pg.logger.Debug("Updated last known price",
		"queryId", queryIdHex,
		"price", price,
	)
}
