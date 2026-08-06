package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/metrics"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/notify"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rules"
)

// Dispatcher sends alerts with rate limiting and dedupe.
// Full alert text is logged independently of Discord delivery so log agents see
// every match; Discord still uses dedupe + rate_limit.
type Dispatcher struct {
	engine      *rules.Engine
	sender      notify.Sender
	limiter     *notify.RateLimiter // Discord
	logLimiter  *notify.RateLimiter // journald (max/window only; no cooldown)
	deduper     *notify.Deduper
	channels    map[string]config.Channel
	logRateLim  config.LogRateLimitConfig
	metrics     *metrics.Registry
	log         *slog.Logger
}

// NewDispatcher constructs an alert dispatcher. metrics may be nil.
func NewDispatcher(
	engine *rules.Engine,
	sender notify.Sender,
	channels map[string]config.Channel,
	logRateLim config.LogRateLimitConfig,
	reg *metrics.Registry,
	log *slog.Logger,
) *Dispatcher {
	if log == nil {
		log = slog.Default()
	}
	d := &Dispatcher{
		engine:     engine,
		sender:     sender,
		limiter:    notify.NewRateLimiter(),
		logLimiter: notify.NewRateLimiter(),
		deduper:    notify.NewDeduper(24 * time.Hour),
		channels:   channels,
		logRateLim: logRateLim,
		metrics:    reg,
		log:        log,
	}
	return d
}

// SendAll delivers alerts, applying Discord dedupe/rate limits. Every match is
// logged (subject to optional log_rate_limit) even when Discord is skipped.
func (d *Dispatcher) SendAll(ctx context.Context, alerts []rules.Alert) error {
	if len(alerts) == 0 {
		return nil
	}
	now := time.Now()
	for _, alert := range alerts {
		msg := notify.Message{
			Content: alert.Content,
			Embeds:  []notify.Embed{alert.Embed},
		}

		discordSkip := ""
		if d.deduper.Has(alert.DedupeKey, now) {
			discordSkip = "dedupe"
			if d.metrics != nil {
				d.metrics.AlertsDeduped.Add(1)
			}
		} else {
			if rl, ok := d.engine.RateLimit(alert.RuleID); ok {
				d.limiter.Configure(alert.RuleID, rl.Max, rl.Window, rl.Cooldown)
			}
			decision := d.limiter.Check(alert.RuleID, now)
			if !decision.Allow {
				discordSkip = "rate_limit"
				if d.metrics != nil {
					d.metrics.AlertsRateLimited.Add(1)
				}
			} else if decision.EnterCooldown {
				msg.Content = fmt.Sprintf("%s\n\n⚠️ **Rate limit reached** — further alerts for `%s` paused until <t:%d:f>.",
					alert.Content, alert.RuleID, decision.CooldownUntil.Unix())
			}
		}

		if discordSkip != "" {
			d.maybeLogAlert(alert, msg, discordSkip, now)
			d.log.Debug("discord skip", "rule", alert.RuleID, "reason", discordSkip, "key", alert.DedupeKey)
			continue
		}

		webhook := ""
		if ch, ok := d.channels[alert.Channel]; ok {
			webhook = ch.WebhookURL
		}
		if err := d.sender.Send(ctx, alert.Channel, webhook, msg); err != nil {
			// Still emit the body so log agents see the alert even when Discord fails.
			d.maybeLogAlert(alert, msg, "send_error", now)
			return fmt.Errorf("send alert %s: %w", alert.RuleID, err)
		}
		d.deduper.Add(alert.DedupeKey, now)
		d.limiter.Record(alert.RuleID, now)
		if d.metrics != nil {
			d.metrics.IncAlertSent(alert.RuleID)
		}
		d.maybeLogAlert(alert, msg, "", now)
	}
	return nil
}

func (d *Dispatcher) maybeLogAlert(alert rules.Alert, msg notify.Message, discordSkip string, now time.Time) {
	d.logLimiter.Configure(alert.RuleID, d.logRateLim.Max, d.logRateLim.Window, 0)
	if !d.logLimiter.AllowWindow(alert.RuleID, now) {
		d.log.Debug("log rate limit skip", "rule", alert.RuleID)
		return
	}

	discord := "sent"
	switch discordSkip {
	case "":
		discord = "sent"
	case "send_error":
		discord = "failed"
	default:
		discord = "skipped"
	}
	attrs := []any{
		"rule", alert.RuleID,
		"channel", alert.Channel,
		"height", alert.Height,
		"discord", discord,
		"text", notify.FormatMessage(msg),
	}
	if discordSkip != "" {
		attrs = append(attrs, "discord_skip", discordSkip)
	}
	d.log.Info("alert", attrs...)
}
