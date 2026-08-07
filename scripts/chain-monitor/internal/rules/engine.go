package rules

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/enrich"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/notify"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/power"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rpc"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/state"
)

// EventSource identifies where an event came from.
type EventSource string

const (
	SourceFinalize EventSource = "finalize_block"
	SourceTx       EventSource = "tx"
	SourceSignal   EventSource = "signal"
)

// EventView is a normalized event for rule evaluation.
type EventView struct {
	Type    string
	Attrs   map[string]string
	Source  EventSource
	TxIndex int // -1 if finalize/signal
}

// BlockView is the input to the rule engine for one height.
type BlockView struct {
	Height  uint64
	Time    time.Time
	ChainID string
	Events  []EventView
}

// Alert is a rule match ready to notify.
type Alert struct {
	RuleID    string
	Channel   string
	Content   string
	Embed     notify.Embed
	DedupeKey string
	Height    uint64
}

// ReportsByAggregateQuerier fetches reporters that submitted for an aggregate.
type ReportsByAggregateQuerier interface {
	ReportsByAggregate(ctx context.Context, queryID string, timestamp uint64) ([]string, error)
}

// EngineOpts configures optional engine dependencies.
type EngineOpts struct {
	QueryIDs  *enrich.QueryIDMap
	Reporters *enrich.ReporterMap
	Important []string
	Oracle    ReportsByAggregateQuerier
	Power     *power.Cache
	Valset    *state.ValsetStore
	Log       *slog.Logger
}

// Engine evaluates configured rules.
type Engine struct {
	nodeName  string
	rules     []compiledRule
	enricher  *enrich.QueryIDMap
	reporters *enrich.ReporterMap
	important []string
	oracle    ReportsByAggregateQuerier
	power     *power.Cache
	valset    *state.ValsetStore
	log       *slog.Logger
}

type compiledRule struct {
	cfg       config.RuleConfig
	rateLimit config.RateLimitConfig
}

// NewEngine builds an engine from config and optional deps.
func NewEngine(cfg *config.Config, opts EngineOpts) *Engine {
	rules := make([]compiledRule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rules = append(rules, compiledRule{
			cfg:       r,
			rateLimit: cfg.RateLimitFor(r),
		})
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	return &Engine{
		nodeName:  cfg.NodeName,
		rules:     rules,
		enricher:  opts.QueryIDs,
		reporters: opts.Reporters,
		important: append([]string(nil), opts.Important...),
		oracle:    opts.Oracle,
		power:     opts.Power,
		valset:    opts.Valset,
		log:       log,
	}
}

// RateLimit returns the rate limit settings for a rule id.
func (e *Engine) RateLimit(ruleID string) (config.RateLimitConfig, bool) {
	for _, r := range e.rules {
		if r.cfg.ID == ruleID {
			return r.rateLimit, true
		}
	}
	return config.RateLimitConfig{}, false
}

// RulesByKind returns rules with the given match.kind.
func (e *Engine) RulesByKind(kind string) []config.RuleConfig {
	kind = strings.ToLower(kind)
	var out []config.RuleConfig
	for _, r := range e.rules {
		if strings.ToLower(strings.TrimSpace(r.cfg.Match.Kind)) == kind {
			out = append(out, r.cfg)
		}
	}
	return out
}

// reportsCache memoizes LCD get_reports_by_aggregate lookups within one Evaluate.
type reportsCache map[string]reportsCacheEntry

type reportsCacheEntry struct {
	reporters []string
	err       error
}

// Evaluate runs event rules against a block (including when predicates + side effects).
// When IMPORTANT_REPORTERS is configured, every aggregate_report is checked and
// missing reporters are logged for AI log review (independent of Discord alerts).
func (e *Engine) Evaluate(block BlockView) []Alert {
	cache := reportsCache{}
	var alerts []Alert
	for _, ev := range block.Events {
		e.logMissingImportantReporters(ev, cache)
		for _, rule := range e.rules {
			if !rule.cfg.IsEventRule() {
				continue
			}
			if !matchEvent(rule.cfg.Match, ev) {
				continue
			}
			if !e.passesWhen(rule.cfg, ev) {
				continue
			}
			e.applySideEffects(rule.cfg, ev)
			alert, ok := e.buildAlert(rule.cfg, block.Height, block.Time, block.ChainID, ev, cache)
			if ok {
				alerts = append(alerts, alert)
			}
		}
	}
	return alerts
}

// EvaluateBlockInterval checks slow-block rules given consecutive block times.
func (e *Engine) EvaluateBlockInterval(prevHeight, height uint64, prevTime, currTime time.Time) []Alert {
	if prevTime.IsZero() || currTime.IsZero() || height <= prevHeight {
		return nil
	}
	interval := currTime.Sub(prevTime)
	blockSpan := height - prevHeight
	normalized := time.Duration(float64(interval) / float64(blockSpan))

	var alerts []Alert
	for _, rule := range e.RulesByKind(config.KindBlockInterval) {
		if normalized <= rule.When.MaxInterval {
			continue
		}
		ev := EventView{
			Type:   config.KindBlockInterval,
			Source: SourceSignal,
			Attrs: map[string]string{
				"interval":            interval.String(),
				"normalized_interval": normalized.String(),
				"max_interval":        rule.When.MaxInterval.String(),
				"prev_height":         strconv.FormatUint(prevHeight, 10),
				"prev_time":           prevTime.UTC().Format(time.RFC3339),
				"curr_time":           currTime.UTC().Format(time.RFC3339),
			},
			TxIndex: -1,
		}
		if alert, ok := e.buildAlert(rule, height, currTime, "", ev, nil); ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// EvaluateRPCUnhealthy fires when consecutive RPC failures reach threshold.
func (e *Engine) EvaluateRPCUnhealthy(consecutiveFails int, lastErr string) []Alert {
	var alerts []Alert
	for _, rule := range e.RulesByKind(config.KindRPCUnhealthy) {
		if consecutiveFails < rule.When.FailThreshold {
			continue
		}
		ev := EventView{
			Type:   config.KindRPCUnhealthy,
			Source: SourceSignal,
			Attrs: map[string]string{
				"consecutive_fails": strconv.Itoa(consecutiveFails),
				"fail_threshold":    strconv.Itoa(rule.When.FailThreshold),
				"last_error":        lastErr,
			},
			TxIndex: -1,
		}
		if alert, ok := e.buildAlert(rule, 0, time.Now().UTC(), "", ev, nil); ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// EvaluateIngestLag fires when tip-cursor lag exceeds max_lag.
func (e *Engine) EvaluateIngestLag(cursor, tip uint64) []Alert {
	if tip < cursor {
		return nil
	}
	lag := tip - cursor
	var alerts []Alert
	for _, rule := range e.RulesByKind(config.KindIngestLag) {
		if lag <= rule.When.MaxLag {
			continue
		}
		ev := EventView{
			Type:   config.KindIngestLag,
			Source: SourceSignal,
			Attrs: map[string]string{
				"cursor":  strconv.FormatUint(cursor, 10),
				"tip":     strconv.FormatUint(tip, 10),
				"lag":     strconv.FormatUint(lag, 10),
				"max_lag": strconv.FormatUint(rule.When.MaxLag, 10),
			},
			TxIndex: -1,
		}
		if alert, ok := e.buildAlert(rule, cursor, time.Now().UTC(), "", ev, nil); ok {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// BuildValsetReport builds the daily valset analysis alert for a schedule rule.
func (e *Engine) BuildValsetReport(rule config.RuleConfig) (Alert, bool) {
	if e.valset == nil || !e.valset.Enabled() {
		ev := EventView{
			Type:   config.KindSchedule,
			Source: SourceSignal,
			Attrs: map[string]string{
				"status": "valset store not configured",
				"count":  "0",
			},
			TxIndex: -1,
		}
		return e.buildAlert(rule, 0, time.Now().UTC(), "", ev, nil)
	}
	timestamps, err := e.valset.Recent(rule.When.Lookback)
	if err != nil {
		ev := EventView{
			Type:   config.KindSchedule,
			Source: SourceSignal,
			Attrs: map[string]string{
				"status": fmt.Sprintf("error reading valset store: %v", err),
				"count":  "0",
			},
			TxIndex: -1,
		}
		return e.buildAlert(rule, 0, time.Now().UTC(), "", ev, nil)
	}
	count, avg, median, latest := state.AnalyzeFrequency(timestamps)
	attrs := map[string]string{
		"count":    strconv.Itoa(count),
		"lookback": rule.When.Lookback.String(),
	}
	switch {
	case count == 0:
		attrs["status"] = "no validator set updates in lookback window"
	case count < 2:
		attrs["status"] = "insufficient updates for frequency analysis"
		attrs["latest"] = latest.Format("2006-01-02 15:04:05 UTC")
	default:
		attrs["status"] = "ok"
		attrs["latest"] = latest.Format("2006-01-02 15:04:05 UTC")
		attrs["average_frequency"] = formatDuration(avg)
		attrs["median_frequency"] = formatDuration(median)
	}
	ev := EventView{Type: config.KindSchedule, Source: SourceSignal, Attrs: attrs, TxIndex: -1}
	return e.buildAlert(rule, 0, time.Now().UTC(), "", ev, nil)
}

func (e *Engine) passesWhen(rule config.RuleConfig, ev EventView) bool {
	w := rule.When.AttrUintLtRatio
	if w == nil {
		return true
	}
	raw := strings.TrimSpace(ev.Attrs[w.Attr])
	if raw == "" {
		return false
	}
	val, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return false
	}
	if w.Against != "validator_power" || e.power == nil {
		return false
	}
	total, ok := e.power.Get()
	if !ok || total == 0 {
		// Skip ratio alerts when power is unknown/stale — never use a fake default.
		return false
	}
	threshold := float64(total) * w.Ratio
	return float64(val) < threshold
}

func (e *Engine) applySideEffects(rule config.RuleConfig, ev EventView) {
	if e.valset == nil || !e.valset.Enabled() {
		return
	}
	for _, se := range rule.SideEffects {
		if se.RecordAttrAsTimestamp == "" {
			continue
		}
		raw := ev.Attrs[se.RecordAttrAsTimestamp]
		if raw == "" {
			continue
		}
		_ = e.valset.Append(raw)
	}
}

func matchEvent(m config.MatchConfig, ev EventView) bool {
	if ev.Type != m.EventType {
		return false
	}
	for _, key := range m.AttrExists {
		if strings.TrimSpace(ev.Attrs[key]) == "" {
			return false
		}
	}
	return true
}

func (e *Engine) buildAlert(rule config.RuleConfig, height uint64, blockTime time.Time, chainID string, ev EventView, cache reportsCache) (Alert, bool) {
	assetPair := ""
	extra := map[string]string{}
	channel := rule.Channel
	for _, kind := range rule.Enrich {
		switch kind {
		case "asset_pair":
			if e.enricher == nil {
				continue
			}
			if qid := ev.Attrs["query_id"]; qid != "" {
				assetPair = e.enricher.AssetPair(qid)
			}
			if assetPair == "" {
				if qd := ev.Attrs["query_data"]; qd != "" {
					assetPair = e.enricher.AssetPairFromQueryData(qd)
				}
			}
		case "missing_reporters":
			if missing := e.resolveMissingReporters(ev, cache); missing != "" {
				extra["_missing_reporters"] = missing
			}
		case "bridge_deposit":
			dep, ok := enrich.DecodeBridgeDepositQueryData(ev.Attrs["query_data"])
			if !ok {
				continue
			}
			extra["_deposit_id"] = strconv.FormatUint(dep.DepositID, 10)
			extra["_query_type"] = dep.QueryType
			if tip := enrich.TipCommand(ev.Attrs["query_data"], chainID); tip != "" {
				extra["_tip_cmd"] = tip
			}
			channel = enrich.BridgeChannel
			if assetPair == "" {
				assetPair = "TRB Bridge Deposit"
			}
		}
	}

	if e.power != nil {
		if p, ok := e.power.Get(); ok {
			extra["_validator_power"] = strconv.FormatUint(p, 10)
			if agg := ev.Attrs["aggregate_power"]; agg != "" {
				if v, err := strconv.ParseUint(agg, 10, 64); err == nil && p > 0 {
					pct := 100 * float64(v) / float64(p)
					extra["_power_pct"] = fmt.Sprintf("%.2f%%", pct)
				}
			}
		}
	}

	ctx := fieldContext{
		attrs:     mergeAttrs(ev.Attrs, extra),
		height:    height,
		blockTime: blockTime,
		node:      e.nodeName,
		source:    string(ev.Source),
		assetPair: assetPair,
	}

	fields := make([]notify.Field, 0, len(rule.Embed.Fields)+8)
	used := make(map[string]struct{})
	for _, f := range rule.Embed.Fields {
		val, ok := resolveAttr(f.Attr, ctx)
		if !ok || val == "" {
			continue
		}
		fields = append(fields, notify.Field{
			Name:   f.Name,
			Value:  formatValue(val, f.Format),
			Inline: f.Inline,
		})
		used[f.Attr] = struct{}{}
	}

	if rule.Embed.IncludeRemaining {
		for k, v := range ev.Attrs {
			if _, seen := used[k]; seen {
				continue
			}
			fields = append(fields, notify.Field{
				Name:   k,
				Value:  formatValue(v, "code"),
				Inline: false,
			})
		}
	}

	ts := blockTime
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	embed := notify.Embed{
		Title:     rule.Embed.Title,
		Color:     rule.Embed.Color,
		Fields:    fields,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Footer: &notify.Footer{
			Text: fmt.Sprintf("%s · height %d", e.nodeName, height),
		},
	}

	return Alert{
		RuleID:    rule.ID,
		Channel:   channel,
		Content:   rule.Embed.Content,
		Embed:     embed,
		DedupeKey: dedupeKey(rule, height, ev),
		Height:    height,
	}, true
}

func mergeAttrs(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

type fieldContext struct {
	attrs     map[string]string
	height    uint64
	blockTime time.Time
	node      string
	source    string
	assetPair string
}

// logMissingImportantReporters checks every aggregate_report against IMPORTANT_REPORTERS
// and emits a structured log when any are absent (for AI log review).
func (e *Engine) logMissingImportantReporters(ev EventView, cache reportsCache) {
	if ev.Type != "aggregate_report" || len(e.important) == 0 || e.oracle == nil {
		return
	}
	missing := e.missingImportantReporters(ev, cache)
	if len(missing) == 0 {
		return
	}
	queryID := strings.TrimSpace(ev.Attrs["query_id"])
	attrs := []any{
		"query_id", queryID,
		"missing_reporters", missing,
	}
	if qt := enrich.QueryTypeFromQueryData(ev.Attrs["query_data"]); qt != "" {
		attrs = append(attrs, "query_type", qt)
	}
	e.log.Info("important reporters missing from aggregate", attrs...)
}

func (e *Engine) resolveMissingReporters(ev EventView, cache reportsCache) string {
	return strings.Join(e.missingImportantReporters(ev, cache), ", ")
}

func (e *Engine) missingImportantReporters(ev EventView, cache reportsCache) []string {
	if len(e.important) == 0 || e.oracle == nil {
		return nil
	}
	submitted, ok := e.reportsByAggregate(ev, cache)
	if !ok {
		return nil
	}
	return enrich.MissingReporters(e.important, submitted, e.reporters)
}

func (e *Engine) reportsByAggregate(ev EventView, cache reportsCache) ([]string, bool) {
	queryID := strings.TrimSpace(ev.Attrs["query_id"])
	timestampStr := strings.TrimSpace(ev.Attrs["timestamp"])
	if queryID == "" || timestampStr == "" {
		e.log.Warn("important reporters check skipped: aggregate event missing query_id or timestamp",
			"query_id", queryID != "",
			"timestamp", timestampStr != "",
		)
		return nil, false
	}
	timestamp, err := strconv.ParseUint(timestampStr, 10, 64)
	if err != nil {
		e.log.Warn("important reporters check skipped: bad timestamp", "timestamp", timestampStr, "err", err)
		return nil, false
	}

	key := queryID + "|" + timestampStr
	if cache != nil {
		if entry, hit := cache[key]; hit {
			if entry.err != nil {
				return nil, false
			}
			return entry.reporters, true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	submitted, err := e.oracle.ReportsByAggregate(ctx, queryID, timestamp)
	if cache != nil {
		cache[key] = reportsCacheEntry{reporters: submitted, err: err}
	}
	if err != nil {
		e.log.Warn("important reporters check failed", "query_id", queryID, "timestamp", timestamp, "err", err)
		return nil, false
	}
	return submitted, true
}

func resolveAttr(attr string, ctx fieldContext) (string, bool) {
	switch attr {
	case "_height":
		if ctx.height == 0 {
			return "", false
		}
		return strconv.FormatUint(ctx.height, 10), true
	case "_time":
		if ctx.blockTime.IsZero() {
			return "", false
		}
		return ctx.blockTime.UTC().Format("2006-01-02 15:04:05"), true
	case "_node":
		return ctx.node, true
	case "_source":
		return ctx.source, true
	case "_asset_pair":
		if ctx.assetPair == "" {
			return "Unknown", true
		}
		return ctx.assetPair, true
	case "_validator_power", "_power_pct", "_missing_reporters", "_deposit_id", "_tip_cmd", "_query_type":
		v, ok := ctx.attrs[attr]
		return v, ok && v != ""
	default:
		v, ok := ctx.attrs[attr]
		return v, ok
	}
}

func formatValue(v, format string) string {
	switch strings.ToLower(format) {
	case "code":
		return "`" + escapeCode(v) + "`"
	case "amount_trb":
		return formatAmountTRB(v)
	default:
		return v
	}
}

func escapeCode(s string) string {
	return strings.ReplaceAll(s, "`", "'")
}

var coinRe = regexp.MustCompile(`(?i)^(-?\d+)([a-zA-Z]+)?$`)

func formatAmountTRB(raw string) string {
	raw = strings.TrimSpace(raw)
	m := coinRe.FindStringSubmatch(raw)
	if m == nil {
		return raw
	}
	amountStr, denom := m[1], strings.ToLower(m[2])
	n := new(big.Int)
	if _, ok := n.SetString(amountStr, 10); !ok {
		return raw
	}
	if denom == "" || denom == "loya" {
		rat := new(big.Rat).SetFrac(n, big.NewInt(1_000_000))
		f, _ := rat.Float64()
		return fmt.Sprintf("%.6f TRB", f)
	}
	return raw
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
}

func dedupeKey(rule config.RuleConfig, height uint64, ev EventView) string {
	if !rule.IsEventRule() {
		return rule.ID + "|" + ev.Type
	}
	parts := []string{rule.ID, strconv.FormatUint(height, 10), ev.Type}
	attrs := rule.DedupeAttrs
	if len(attrs) == 0 {
		for _, f := range rule.Embed.Fields {
			if !strings.HasPrefix(f.Attr, "_") {
				attrs = append(attrs, f.Attr)
			}
		}
	}
	for _, a := range attrs {
		parts = append(parts, a+"="+ev.Attrs[a])
	}
	return strings.Join(parts, "|")
}

// BlockViewFromRPC converts an RPC block result into a BlockView.
// Events that appear in both finalize_block_events and txs_results (identical
// type + all attributes) are kept once — finalize wins — so rules do not fire
// twice for the same emission. Distinct events that only share some attrs are
// preserved; only exact cross-array copies are dropped.
func BlockViewFromRPC(block *rpc.BlockResult) BlockView {
	bv := BlockView{
		Height:  block.Height,
		ChainID: block.ChainID,
	}
	if block.Time != "" {
		if t, err := time.Parse(time.RFC3339Nano, block.Time); err == nil {
			bv.Time = t
		} else if t, err := time.Parse(time.RFC3339, block.Time); err == nil {
			bv.Time = t
		}
	}

	finalizeFP := make(map[string]struct{}, len(block.FinalizeBlockEvents))
	for _, ev := range block.FinalizeBlockEvents {
		attrs := attrsToMap(ev.Attributes)
		fp := eventFingerprint(ev.Type, attrs)
		finalizeFP[fp] = struct{}{}
		bv.Events = append(bv.Events, EventView{
			Type:    ev.Type,
			Attrs:   attrs,
			Source:  SourceFinalize,
			TxIndex: -1,
		})
	}
	for i, tx := range block.TxsResults {
		for _, ev := range tx.Events {
			attrs := attrsToMap(ev.Attributes)
			fp := eventFingerprint(ev.Type, attrs)
			if _, dup := finalizeFP[fp]; dup {
				continue
			}
			bv.Events = append(bv.Events, EventView{
				Type:    ev.Type,
				Attrs:   attrs,
				Source:  SourceTx,
				TxIndex: i,
			})
		}
	}
	return bv
}

// eventFingerprint is a stable identity for an ABCI event: type + every attribute.
func eventFingerprint(eventType string, attrs map[string]string) string {
	if len(attrs) == 0 {
		return eventType
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(eventType)
	for _, k := range keys {
		b.WriteByte('|')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(attrs[k])
	}
	return b.String()
}

func attrsToMap(attrs []rpc.Attribute) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, a := range attrs {
		out[a.Key] = a.Value
	}
	return out
}
