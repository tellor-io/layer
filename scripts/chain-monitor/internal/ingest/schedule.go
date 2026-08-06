package ingest

import (
	"context"
	"log/slog"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rules"
)

// RunSchedules runs daily_at schedule rules until ctx is cancelled.
func RunSchedules(ctx context.Context, engine *rules.Engine, dispatcher *Dispatcher, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	rules := engine.RulesByKind(config.KindSchedule)
	if len(rules) == 0 {
		return
	}
	for _, rule := range rules {
		r := rule
		go runOneSchedule(ctx, r, engine, dispatcher, log)
	}
	<-ctx.Done()
}

func runOneSchedule(ctx context.Context, rule config.RuleConfig, engine *rules.Engine, dispatcher *Dispatcher, log *slog.Logger) {
	hour, min, err := config.ParseDailyAt(rule.When.DailyAt)
	if err != nil {
		log.Error("invalid schedule rule", "rule", rule.ID, "err", err)
		return
	}
	log.Info("schedule rule armed", "rule", rule.ID, "daily_at", rule.When.DailyAt)

	for {
		wait := timeUntilDaily(time.Now(), hour, min)
		log.Info("schedule sleeping until next run", "rule", rule.ID, "wait", wait.Round(time.Second))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		alert, ok := engine.BuildValsetReport(rule)
		if !ok {
			continue
		}
		if err := dispatcher.SendAll(ctx, []rules.Alert{alert}); err != nil {
			log.Warn("schedule alert failed", "rule", rule.ID, "err", err)
		} else {
			log.Info("schedule alert sent", "rule", rule.ID)
		}
	}
}

func timeUntilDaily(now time.Time, hour, min int) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}
