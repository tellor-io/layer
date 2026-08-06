package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/health"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/metrics"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/notify"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rpc"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rules"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/state"
)

// Config holds poller runtime settings.
type Config struct {
	NodeName     string
	StartHeight  uint64
	Channels     map[string]config.Channel
	LogRateLimit config.LogRateLimitConfig
}

// Poller walks the chain height-by-height, evaluates rules, and advances the cursor.
type Poller struct {
	client     *rpc.Client
	cursor     *state.CursorFile
	health     *health.Tracker
	engine     *rules.Engine
	dispatcher *Dispatcher
	metrics    *metrics.Registry
	log        *slog.Logger
	cfg        Config

	prevBlockTime     time.Time
	prevBlockHeight   uint64
	rpcFails          int
	lastLagCheck      time.Time
	lastRPCAlertCheck time.Time
}

// NewPoller constructs a poller. metrics may be nil.
func NewPoller(
	client *rpc.Client,
	cursor *state.CursorFile,
	tracker *health.Tracker,
	engine *rules.Engine,
	sender notify.Sender,
	reg *metrics.Registry,
	log *slog.Logger,
	cfg Config,
) *Poller {
	if log == nil {
		log = slog.Default()
	}
	return &Poller{
		client:     client,
		cursor:     cursor,
		health:     tracker,
		engine:     engine,
		dispatcher: NewDispatcher(engine, sender, cfg.Channels, cfg.LogRateLimit, reg, log),
		metrics:    reg,
		log:        log,
		cfg:        cfg,
	}
}

// Dispatcher exposes the shared alert dispatcher (for schedule runners).
func (p *Poller) Dispatcher() *Dispatcher {
	return p.dispatcher
}

// Run resolves the starting height and processes blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	height, err := p.resolveStartHeight(ctx)
	if err != nil {
		return err
	}
	p.log.Info("ingest starting", "height", height, "cursor_path", p.cursor.Path())
	p.health.SetReady(true)

	var (
		waitNotReady = 500 * time.Millisecond
		errBackoff   = time.Second
		maxBackoff   = 30 * time.Second
	)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		tip, tipErr := p.client.LatestHeight(ctx)
		if tipErr != nil {
			p.noteRPCFailure(ctx, tipErr)
			p.health.RecordError(tipErr)
			if p.metrics != nil {
				p.metrics.RPCErrors.Add(1)
			}
			p.log.Warn("failed to fetch tip height", "err", tipErr)
			if !sleep(ctx, errBackoff) {
				return ctx.Err()
			}
			errBackoff = nextBackoff(errBackoff, maxBackoff)
			continue
		}
		errBackoff = time.Second
		p.noteRPCSuccess()

		var cursor uint64
		if height > 0 {
			cursor = height - 1
		}
		if err := p.maybeCheckLag(ctx, cursor, tip); err != nil {
			p.log.Warn("ingest lag alert failed", "err", err)
		}

		if height > tip {
			if !sleep(ctx, waitNotReady) {
				return ctx.Err()
			}
			continue
		}

		block, err := p.client.GetBlockResult(ctx, height)
		if err != nil {
			if errors.Is(err, rpc.ErrBlockNotFound) {
				if !sleep(ctx, waitNotReady) {
					return ctx.Err()
				}
				continue
			}
			p.noteRPCFailure(ctx, err)
			p.health.RecordError(err)
			if p.metrics != nil {
				p.metrics.RPCErrors.Add(1)
			}
			p.log.Warn("failed to fetch block", "height", height, "err", err)
			if !sleep(ctx, errBackoff) {
				return ctx.Err()
			}
			errBackoff = nextBackoff(errBackoff, maxBackoff)
			continue
		}
		errBackoff = time.Second
		p.noteRPCSuccess()

		if err := p.processBlock(ctx, block); err != nil {
			p.health.RecordError(err)
			p.log.Warn("failed to process block", "height", height, "err", err)
			if !sleep(ctx, errBackoff) {
				return ctx.Err()
			}
			errBackoff = nextBackoff(errBackoff, maxBackoff)
			continue
		}

		if err := p.cursor.Save(height); err != nil {
			p.health.RecordError(err)
			p.log.Error("failed to save cursor", "height", height, "err", err)
			if !sleep(ctx, errBackoff) {
				return ctx.Err()
			}
			errBackoff = nextBackoff(errBackoff, maxBackoff)
			continue
		}

		p.health.RecordSuccess(height, tip)
		if p.metrics != nil {
			p.metrics.BlocksProcessed.Add(1)
			p.metrics.SetHeights(height, tip)
		}
		if height%100 == 0 || tip-height < 5 {
			p.log.Info("processed block", "height", height, "tip", tip, "lag", tip-height)
		}

		height++
	}
}

func (p *Poller) noteRPCFailure(ctx context.Context, err error) {
	p.rpcFails++
	// Throttle signal evaluation by the shortest check_every among rpc_unhealthy rules.
	checkEvery := 30 * time.Second
	for _, rule := range p.engine.RulesByKind(config.KindRPCUnhealthy) {
		if rule.When.CheckEvery > 0 && rule.When.CheckEvery < checkEvery {
			checkEvery = rule.When.CheckEvery
		}
	}
	if time.Since(p.lastRPCAlertCheck) < checkEvery {
		return
	}
	p.lastRPCAlertCheck = time.Now()
	alerts := p.engine.EvaluateRPCUnhealthy(p.rpcFails, err.Error())
	if len(alerts) == 0 {
		return
	}
	if sendErr := p.dispatcher.SendAll(ctx, alerts); sendErr != nil {
		p.log.Warn("rpc unhealthy alert failed", "err", sendErr)
	}
}

func (p *Poller) noteRPCSuccess() {
	p.rpcFails = 0
}

func (p *Poller) maybeCheckLag(ctx context.Context, cursor, tip uint64) error {
	checkEvery := 30 * time.Second
	for _, rule := range p.engine.RulesByKind(config.KindIngestLag) {
		if rule.When.CheckEvery > 0 && rule.When.CheckEvery < checkEvery {
			checkEvery = rule.When.CheckEvery
		}
	}
	if time.Since(p.lastLagCheck) < checkEvery {
		return nil
	}
	p.lastLagCheck = time.Now()
	alerts := p.engine.EvaluateIngestLag(cursor, tip)
	if len(alerts) == 0 {
		return nil
	}
	p.log.Info("ingest lag rule matched", "cursor", cursor, "tip", tip, "lag", tip-cursor)
	return p.dispatcher.SendAll(ctx, alerts)
}

func (p *Poller) resolveStartHeight(ctx context.Context) (uint64, error) {
	stored, exists, err := p.cursor.Load()
	if err != nil {
		return 0, fmt.Errorf("load cursor: %w", err)
	}
	if exists {
		next := stored + 1
		p.log.Info("resuming from cursor", "last_processed", stored, "next", next)
		p.health.RecordSuccess(stored, stored)
		return next, nil
	}

	var tip uint64
	errBackoff := time.Second
	for {
		var tipErr error
		tip, tipErr = p.client.LatestHeight(ctx)
		if tipErr == nil {
			break
		}
		p.health.RecordError(tipErr)
		p.log.Warn("waiting for rpc tip before seeding cursor", "err", tipErr)
		if !sleep(ctx, errBackoff) {
			return 0, ctx.Err()
		}
		errBackoff = nextBackoff(errBackoff, 30*time.Second)
	}

	var start uint64
	switch {
	case p.cfg.StartHeight > 0:
		start = p.cfg.StartHeight
		p.log.Info("no cursor found; starting at configured height", "height", start, "tip", tip)
	default:
		start = tip
		p.log.Info("no cursor found; starting at tip", "height", start)
	}

	var lastProcessed uint64
	if start > 0 {
		lastProcessed = start - 1
	}
	if err := p.cursor.Save(lastProcessed); err != nil {
		return 0, fmt.Errorf("seed cursor: %w", err)
	}
	p.health.RecordSuccess(lastProcessed, tip)
	return start, nil
}

func (p *Poller) processBlock(ctx context.Context, block *rpc.BlockResult) error {
	if p.engine == nil {
		return nil
	}

	view := rules.BlockViewFromRPC(block)

	var alerts []rules.Alert
	alerts = append(alerts, p.engine.Evaluate(view)...)

	if !view.Time.IsZero() {
		if !p.prevBlockTime.IsZero() {
			alerts = append(alerts, p.engine.EvaluateBlockInterval(
				p.prevBlockHeight, view.Height, p.prevBlockTime, view.Time,
			)...)
		}
		p.prevBlockTime = view.Time
		p.prevBlockHeight = view.Height
	}

	if len(alerts) == 0 {
		return nil
	}
	p.log.Info("rule matches", "height", block.Height, "alerts", len(alerts))
	return p.dispatcher.SendAll(ctx, alerts)
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}
