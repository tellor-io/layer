package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Registry holds process counters and gauges for Prometheus text exposition.
type Registry struct {
	BlocksProcessed   atomic.Uint64
	RPCErrors         atomic.Uint64
	AlertsSent        atomic.Uint64
	AlertsDeduped     atomic.Uint64
	AlertsRateLimited atomic.Uint64

	mu             sync.RWMutex
	alertsByRule   map[string]uint64
	cursorHeight   uint64
	tipHeight      uint64
	validatorPower uint64
	powerKnown     bool
}

// New creates an empty metrics registry.
func New() *Registry {
	return &Registry{alertsByRule: make(map[string]uint64)}
}

// IncAlertSent increments sent counters, optionally labeled by rule.
func (r *Registry) IncAlertSent(ruleID string) {
	r.AlertsSent.Add(1)
	if ruleID == "" {
		return
	}
	r.mu.Lock()
	r.alertsByRule[ruleID]++
	r.mu.Unlock()
}

// SetHeights updates cursor/tip gauges.
func (r *Registry) SetHeights(cursor, tip uint64) {
	r.mu.Lock()
	r.cursorHeight = cursor
	r.tipHeight = tip
	r.mu.Unlock()
}

// SetValidatorPower updates the validator power gauge.
func (r *Registry) SetValidatorPower(power uint64, known bool) {
	r.mu.Lock()
	r.validatorPower = power
	r.powerKnown = known
	r.mu.Unlock()
}

// Handler returns an http.Handler that serves Prometheus text format.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(r.Expose()))
	})
}

// Expose renders Prometheus text exposition format.
func (r *Registry) Expose() string {
	r.mu.RLock()
	cursor := r.cursorHeight
	tip := r.tipHeight
	power := r.validatorPower
	powerKnown := r.powerKnown
	byRule := make(map[string]uint64, len(r.alertsByRule))
	for k, v := range r.alertsByRule {
		byRule[k] = v
	}
	r.mu.RUnlock()

	var b strings.Builder
	metric(&b, "counter", "chain_monitor_blocks_processed_total",
		"Blocks successfully processed", nil, float64(r.BlocksProcessed.Load()))
	metric(&b, "counter", "chain_monitor_rpc_errors_total",
		"RPC failures observed by the poller", nil, float64(r.RPCErrors.Load()))
	metric(&b, "counter", "chain_monitor_alerts_sent_total",
		"Alerts delivered (or dry-run logged)", nil, float64(r.AlertsSent.Load()))
	metric(&b, "counter", "chain_monitor_alerts_deduped_total",
		"Alerts skipped by dedupe", nil, float64(r.AlertsDeduped.Load()))
	metric(&b, "counter", "chain_monitor_alerts_rate_limited_total",
		"Alerts skipped by rate limit", nil, float64(r.AlertsRateLimited.Load()))

	fmt.Fprintf(&b, "# HELP chain_monitor_alerts_sent_by_rule_total Alerts delivered by rule id\n")
	fmt.Fprintf(&b, "# TYPE chain_monitor_alerts_sent_by_rule_total counter\n")
	if len(byRule) > 0 {
		keys := make([]string, 0, len(byRule))
		for k := range byRule {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "chain_monitor_alerts_sent_by_rule_total{rule=%q} %d\n", k, byRule[k])
		}
	}

	metric(&b, "gauge", "chain_monitor_cursor_height",
		"Last processed block height", nil, float64(cursor))
	metric(&b, "gauge", "chain_monitor_tip_height",
		"Latest known chain tip height", nil, float64(tip))

	var lag uint64
	if tip > cursor {
		lag = tip - cursor
	}
	metric(&b, "gauge", "chain_monitor_ingest_lag",
		"Tip minus cursor height", nil, float64(lag))

	powerVal := float64(0)
	if powerKnown {
		powerVal = float64(power)
	}
	metric(&b, "gauge", "chain_monitor_validator_power",
		"Cached total validator voting power (0 if unknown)", nil, powerVal)

	return b.String()
}

func metric(b *strings.Builder, typ, name, help string, labels map[string]string, v float64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", name, typ)
	if len(labels) == 0 {
		fmt.Fprintf(b, "%s %v\n", name, formatFloat(v))
		return
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	fmt.Fprintf(b, "%s{%s} %v\n", name, strings.Join(parts, ","), formatFloat(v))
}

func formatFloat(v float64) string {
	if v == float64(uint64(v)) {
		return fmt.Sprintf("%d", uint64(v))
	}
	return fmt.Sprintf("%g", v)
}
