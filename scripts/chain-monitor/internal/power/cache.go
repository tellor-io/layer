package power

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/metrics"
)

// Fetcher retrieves total validator voting power.
type Fetcher interface {
	TotalVotingPower(ctx context.Context) (uint64, error)
}

// Cache holds the latest total validator voting power.
// If never successfully refreshed, Get returns ok=false (do not use a fake default).
type Cache struct {
	fetcher    Fetcher
	interval   time.Duration
	staleAfter time.Duration
	metrics    *metrics.Registry
	log        *slog.Logger

	mu        sync.RWMutex
	power     uint64
	updatedAt time.Time
	ok        bool
}

// NewCache creates a power cache. staleAfter defaults to 3x refresh interval. metrics may be nil.
func NewCache(fetcher Fetcher, refreshInterval time.Duration, reg *metrics.Registry, log *slog.Logger) *Cache {
	if log == nil {
		log = slog.Default()
	}
	if refreshInterval <= 0 {
		refreshInterval = 10 * time.Minute
	}
	return &Cache{
		fetcher:    fetcher,
		interval:   refreshInterval,
		staleAfter: refreshInterval * 3,
		metrics:    reg,
		log:        log,
	}
}

// Get returns current power. ok is false if never fetched or stale.
func (c *Cache) Get() (power uint64, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.ok {
		return 0, false
	}
	if c.staleAfter > 0 && time.Since(c.updatedAt) > c.staleAfter {
		return c.power, false
	}
	return c.power, true
}

// Refresh fetches and stores power once.
func (c *Cache) Refresh(ctx context.Context) error {
	p, err := c.fetcher.TotalVotingPower(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.power = p
	c.updatedAt = time.Now().UTC()
	c.ok = true
	c.mu.Unlock()
	if c.metrics != nil {
		c.metrics.SetValidatorPower(p, true)
	}
	c.log.Info("updated validator voting power", "total", p)
	return nil
}

// Start runs periodic refreshes until ctx is done. Performs an immediate refresh first.
func (c *Cache) Start(ctx context.Context) {
	if err := c.Refresh(ctx); err != nil {
		c.log.Warn("initial power refresh failed", "err", err)
	}
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.Refresh(ctx); err != nil {
				c.log.Warn("power refresh failed", "err", err)
			}
		}
	}
}
