package rules

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/enrich"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/power"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rpc"
)

type staticPower struct{ p uint64 }

func (s staticPower) TotalVotingPower(context.Context) (uint64, error) { return s.p, nil }

func TestEvaluateDepositClaimed(t *testing.T) {
	cfg := &config.Config{
		NodeName: "test-node",
		Channels: map[string]config.Channel{
			"bridge": {WebhookURL: "https://example.com"},
		},
		Rules: []config.RuleConfig{
			{
				ID:      "deposit_claimed",
				Channel: "bridge",
				Match:   config.MatchConfig{EventType: "deposit_claimed"},
				Embed: config.EmbedConfig{
					Content: "🚨 **New Bridge Deposit Detected!**",
					Title:   "🌉 New Bridge Deposit",
					Color:   0x2ECC71,
					Fields: []config.FieldConfig{
						{Attr: "deposit_id", Name: "📍 Deposit ID", Inline: true, Format: "code"},
						{Attr: "_time", Name: "⏰ Timestamp", Inline: true},
						{Attr: "recipient", Name: "🎯 Recipient", Inline: false, Format: "code"},
						{Attr: "amount", Name: "💰 Amount", Inline: true, Format: "amount_trb"},
						{Attr: "_height", Name: "🧱 Block Height", Inline: true, Format: "code"},
					},
				},
			},
		},
	}
	cfg.Defaults.RateLimit.Max = 10
	cfg.Defaults.RateLimit.Window = time.Minute
	cfg.Defaults.RateLimit.Cooldown = time.Hour

	engine := NewEngine(cfg, EngineOpts{})
	block := BlockView{
		Height: 25669427,
		Time:   time.Date(2026, 8, 2, 15, 59, 45, 0, time.UTC),
		Events: []EventView{
			{
				Type:   "deposit_claimed",
				Source: SourceFinalize,
				Attrs: map[string]string{
					"deposit_id": "206",
					"recipient":  "tellor128m9knt3039k5rmaeu50q0g7y608g2w5etj2ra",
					"amount":     "5092440000loya",
				},
			},
		},
	}

	alerts := engine.Evaluate(block)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	if a.Channel != "bridge" || a.Embed.Title != "🌉 New Bridge Deposit" {
		t.Fatalf("unexpected alert: %+v", a)
	}
	foundAmount := false
	for _, f := range a.Embed.Fields {
		if f.Name == "💰 Amount" {
			foundAmount = true
			if f.Value != "5092.440000 TRB" {
				t.Fatalf("amount = %q", f.Value)
			}
		}
	}
	if !foundAmount {
		t.Fatal("missing amount field")
	}
}

func TestWeakAggregateRatio(t *testing.T) {
	cache := power.NewCache(staticPower{p: 1000}, time.Minute, nil, nil)
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"oracle": {WebhookURL: "http://x"}},
		Rules: []config.RuleConfig{{
			ID:      "weak_aggregate",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "aggregate_report"},
			When: config.WhenConfig{
				AttrUintLtRatio: &config.AttrUintLtRatio{
					Attr:    "aggregate_power",
					Ratio:   2.0 / 3.0,
					Against: "validator_power",
				},
			},
			Embed: config.EmbedConfig{
				Title: "Weak Aggregate",
				Fields: []config.FieldConfig{
					{Attr: "aggregate_power", Name: "Power"},
					{Attr: "_power_pct", Name: "Pct"},
				},
			},
		}},
	}
	engine := NewEngine(cfg, EngineOpts{Power: cache})

	// 500 < 666.7 → alert
	alerts := engine.Evaluate(BlockView{
		Height: 1,
		Events: []EventView{{
			Type:  "aggregate_report",
			Attrs: map[string]string{"aggregate_power": "500", "query_id": "abc"},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected weak alert, got %d", len(alerts))
	}

	// 800 >= 666.7 → no alert
	alerts = engine.Evaluate(BlockView{
		Height: 2,
		Events: []EventView{{
			Type:  "aggregate_report",
			Attrs: map[string]string{"aggregate_power": "800"},
		}},
	})
	if len(alerts) != 0 {
		t.Fatalf("expected no alert for strong aggregate, got %d", len(alerts))
	}
}

func TestWeakAggregateBridgeDepositRoutesToBridge(t *testing.T) {
	cache := power.NewCache(staticPower{p: 1000}, time.Minute, nil, nil)
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	const queryData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b54524242726964676556320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000ce"

	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{
			"oracle": {WebhookURL: "http://x"},
			"bridge": {WebhookURL: "http://y"},
		},
		Rules: []config.RuleConfig{{
			ID:      "weak_aggregate",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "aggregate_report"},
			When: config.WhenConfig{
				AttrUintLtRatio: &config.AttrUintLtRatio{
					Attr: "aggregate_power", Ratio: 0.666, Against: "validator_power",
				},
			},
			Enrich: []string{"bridge_deposit"},
			Embed: config.EmbedConfig{
				Title: "Weak Aggregate",
				Fields: []config.FieldConfig{
					{Attr: "_asset_pair", Name: "Asset"},
					{Attr: "_deposit_id", Name: "Deposit ID", Format: "code"},
					{Attr: "_tip_cmd", Name: "Tip", Format: "code"},
					{Attr: "aggregate_power", Name: "Power"},
				},
			},
		}},
	}
	engine := NewEngine(cfg, EngineOpts{Power: cache})

	alerts := engine.Evaluate(BlockView{
		Height:  42,
		ChainID: "layertest-4",
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"aggregate_power":   "500",
				"query_data":        queryData,
				"micro_report_type": "TRBBridgeV2",
			},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	if a.Channel != "bridge" {
		t.Fatalf("channel = %q want bridge", a.Channel)
	}
	fields := map[string]string{}
	for _, f := range a.Embed.Fields {
		fields[f.Name] = f.Value
	}
	if fields["Deposit ID"] != "`206`" {
		t.Fatalf("deposit id = %q", fields["Deposit ID"])
	}
	if fields["Asset"] != "TRB Bridge Deposit" {
		t.Fatalf("asset = %q", fields["Asset"])
	}
	wantTip := "`./layerd tx oracle tip " + queryData + " 1500loya --chain-id layertest-4`"
	if fields["Tip"] != wantTip {
		t.Fatalf("tip =\n%q\nwant\n%q", fields["Tip"], wantTip)
	}

	// Non-bridge weak aggregate stays on oracle and omits deposit/tip fields.
	alerts = engine.Evaluate(BlockView{
		Height:  43,
		ChainID: "layertest-4",
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"aggregate_power": "500",
				"query_data":      "aabbcc",
			},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected spotprice alert, got %d", len(alerts))
	}
	if alerts[0].Channel != "oracle" {
		t.Fatalf("channel = %q want oracle", alerts[0].Channel)
	}
	for _, f := range alerts[0].Embed.Fields {
		if f.Name == "Deposit ID" || f.Name == "Tip" {
			t.Fatalf("unexpected bridge field %q on non-bridge alert", f.Name)
		}
	}
}

func TestWeakAggregateSkippedWithoutPower(t *testing.T) {
	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"oracle": {WebhookURL: "http://x"}},
		Rules: []config.RuleConfig{{
			ID:      "weak_aggregate",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "aggregate_report"},
			When: config.WhenConfig{
				AttrUintLtRatio: &config.AttrUintLtRatio{
					Attr: "aggregate_power", Ratio: 0.666, Against: "validator_power",
				},
			},
			Embed: config.EmbedConfig{Title: "Weak"},
		}},
	}
	engine := NewEngine(cfg, EngineOpts{}) // no power cache
	alerts := engine.Evaluate(BlockView{
		Height: 1,
		Events: []EventView{{Type: "aggregate_report", Attrs: map[string]string{"aggregate_power": "1"}}},
	})
	if len(alerts) != 0 {
		t.Fatalf("expected skip without power, got %d", len(alerts))
	}
}

type stubOracle struct {
	reporters []string
	err       error
	calls     int
}

func (s *stubOracle) ReportsByAggregate(context.Context, string, uint64) ([]string, error) {
	s.calls++
	return s.reporters, s.err
}

func TestMissingReportersEnrich(t *testing.T) {
	cache := power.NewCache(staticPower{p: 1000}, time.Minute, nil, nil)
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := dir + "/reporters.json"
	if err := os.WriteFile(path, []byte(`{"addressToMonikerMap":{"tellor1a":"Alice","tellor1b":"Bob"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	rm, err := enrich.NewReporterMap(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"oracle": {WebhookURL: "http://x"}},
		Rules: []config.RuleConfig{{
			ID:      "weak_aggregate",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "aggregate_report"},
			When: config.WhenConfig{
				AttrUintLtRatio: &config.AttrUintLtRatio{
					Attr: "aggregate_power", Ratio: 0.666, Against: "validator_power",
				},
			},
			Enrich: []string{"missing_reporters"},
			Embed: config.EmbedConfig{
				Title: "Weak Aggregate",
				Fields: []config.FieldConfig{
					{Attr: "aggregate_power", Name: "Power"},
					{Attr: "_missing_reporters", Name: "Missing"},
				},
			},
		}},
	}

	oracle := &stubOracle{reporters: []string{"tellor1a"}}
	engine := NewEngine(cfg, EngineOpts{
		Power:     cache,
		Reporters: rm,
		Important: []string{"tellor1a", "tellor1b"},
		Oracle:    oracle,
	})
	alerts := engine.Evaluate(BlockView{
		Height: 1,
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"aggregate_power": "500",
				"query_id":        "abc",
				"timestamp":       "12345",
			},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	var missing string
	for _, f := range alerts[0].Embed.Fields {
		if f.Name == "Missing" {
			missing = f.Value
		}
	}
	if missing != "Bob" {
		t.Fatalf("missing field = %q want Bob", missing)
	}
	// Log check + enrich share one LCD fetch.
	if oracle.calls != 1 {
		t.Fatalf("oracle calls = %d want 1", oracle.calls)
	}

	// LCD error: alert still fires, missing field omitted.
	oracle = &stubOracle{err: context.DeadlineExceeded}
	engine = NewEngine(cfg, EngineOpts{
		Power:     cache,
		Reporters: rm,
		Important: []string{"tellor1a"},
		Oracle:    oracle,
	})
	alerts = engine.Evaluate(BlockView{
		Height: 2,
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"aggregate_power": "500",
				"query_id":        "abc",
				"timestamp":       "12345",
			},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("expected alert despite oracle error, got %d", len(alerts))
	}
	for _, f := range alerts[0].Embed.Fields {
		if f.Name == "Missing" {
			t.Fatalf("expected no Missing field, got %q", f.Value)
		}
	}
}

func TestImportantReportersLoggedOnEveryAggregate(t *testing.T) {
	var records []map[string]any
	handler := &captureHandler{records: &records}
	log := slog.New(handler)

	oracle := &stubOracle{reporters: []string{"tellor1a"}}
	// No weak_aggregate rule — still check every aggregate for logging.
	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"oracle": {WebhookURL: "http://x"}},
		Rules:    nil,
	}
	engine := NewEngine(cfg, EngineOpts{
		Important: []string{"tellor1a", "tellor1b"},
		Oracle:    oracle,
		Log:       log,
	})
	queryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b54524242726964676556320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000ce"
	alerts := engine.Evaluate(BlockView{
		Height: 10,
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"aggregate_power": "900",
				"query_id":        "qid1",
				"timestamp":       "99",
				"query_data":      queryData,
			},
		}},
	})
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %d", len(alerts))
	}
	if oracle.calls != 1 {
		t.Fatalf("oracle calls = %d", oracle.calls)
	}
	found := false
	for _, rec := range records {
		if rec["msg"] != "important reporters missing from aggregate" {
			continue
		}
		found = true
		if rec["query_id"] != "qid1" {
			t.Fatalf("query_id = %v", rec["query_id"])
		}
		if rec["query_type"] != "TRBBridgeV2" {
			t.Fatalf("query_type = %v", rec["query_type"])
		}
		missing, ok := rec["missing_reporters"].([]string)
		if !ok || len(missing) != 1 || missing[0] != "tellor1b" {
			t.Fatalf("missing_reporters = %#v", rec["missing_reporters"])
		}
	}
	if !found {
		t.Fatalf("expected missing-reporters log, got %#v", records)
	}

	// All important reporters present → no log.
	records = nil
	oracle = &stubOracle{reporters: []string{"tellor1a", "tellor1b"}}
	engine = NewEngine(cfg, EngineOpts{
		Important: []string{"tellor1a", "tellor1b"},
		Oracle:    oracle,
		Log:       slog.New(&captureHandler{records: &records}),
	})
	_ = engine.Evaluate(BlockView{
		Height: 11,
		Events: []EventView{{
			Type: "aggregate_report",
			Attrs: map[string]string{
				"query_id":  "qid2",
				"timestamp": "100",
			},
		}},
	})
	for _, rec := range records {
		if rec["msg"] == "important reporters missing from aggregate" {
			t.Fatal("expected no missing-reporters log when all present")
		}
	}
}

type captureHandler struct {
	records *[]map[string]any
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *captureHandler) WithGroup(string) slog.Handler            { return h }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	m := map[string]any{"msg": r.Message, "level": r.Level.String()}
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})
	*h.records = append(*h.records, m)
	return nil
}

func TestBlockInterval(t *testing.T) {
	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"infra": {WebhookURL: "http://x"}},
		Rules: []config.RuleConfig{{
			ID:      "slow_blocks",
			Channel: "infra",
			Match:   config.MatchConfig{Kind: config.KindBlockInterval},
			When:    config.WhenConfig{MaxInterval: time.Minute},
			Embed: config.EmbedConfig{
				Title:  "Slow Blocks",
				Fields: []config.FieldConfig{{Attr: "normalized_interval", Name: "Interval"}},
			},
		}},
	}
	engine := NewEngine(cfg, EngineOpts{})
	t0 := time.Now()
	alerts := engine.EvaluateBlockInterval(10, 11, t0, t0.Add(2*time.Minute))
	if len(alerts) != 1 {
		t.Fatalf("expected slow block alert, got %d", len(alerts))
	}
	alerts = engine.EvaluateBlockInterval(11, 12, t0, t0.Add(30*time.Second))
	if len(alerts) != 0 {
		t.Fatalf("expected no alert, got %d", len(alerts))
	}
}

func TestFormatAmountTRB(t *testing.T) {
	cases := map[string]string{
		"5092440000loya": "5092.440000 TRB",
		"1000000loya":    "1.000000 TRB",
		"1000000":        "1.000000 TRB",
		"not-a-coin":     "not-a-coin",
	}
	for in, want := range cases {
		if got := formatAmountTRB(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestAssetPairEnrichment(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/map.json"
	content := `{"queryIdToAssetPairMap":{"abc123":"ETH/USD"},"queryDataToAssetPairMap":{}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	qm, err := enrich.NewQueryIDMap(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{"oracle": {WebhookURL: "http://x"}},
		Rules: []config.RuleConfig{{
			ID:      "tip",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "tip_added"},
			Enrich:  []string{"asset_pair"},
			Embed: config.EmbedConfig{
				Title: "Tip",
				Fields: []config.FieldConfig{
					{Attr: "query_id", Name: "Query ID", Format: "code"},
					{Attr: "_asset_pair", Name: "Asset Pair"},
				},
			},
		}},
	}
	engine := NewEngine(cfg, EngineOpts{QueryIDs: qm})
	alerts := engine.Evaluate(BlockView{
		Height: 1,
		Events: []EventView{{
			Type:  "tip_added",
			Attrs: map[string]string{"query_id": "0xABC123"},
		}},
	})
	if len(alerts) != 1 {
		t.Fatalf("alerts=%d", len(alerts))
	}
	for _, f := range alerts[0].Embed.Fields {
		if f.Name == "Asset Pair" && f.Value != "ETH/USD" {
			t.Fatalf("asset pair = %q", f.Value)
		}
	}
}

func TestBlockViewFromRPCCrossArrayDedupe(t *testing.T) {
	block := &rpc.BlockResult{
		Height:  100,
		ChainID: "layer",
		Time:    "2026-08-06T12:00:00Z",
		FinalizeBlockEvents: []rpc.Event{
			{
				Type: "new_dispute",
				Attributes: []rpc.Attribute{
					{Key: "dispute_id", Value: "1"},
					{Key: "reporter", Value: "addrA"},
				},
			},
			{
				Type: "new_dispute",
				Attributes: []rpc.Attribute{
					{Key: "dispute_id", Value: "2"},
					{Key: "reporter", Value: "addrB"},
				},
			},
		},
		TxsResults: []rpc.TxResult{
			{
				Events: []rpc.Event{
					// Exact copy of finalize dispute 1 — should be dropped.
					{
						Type: "new_dispute",
						Attributes: []rpc.Attribute{
							{Key: "reporter", Value: "addrA"}, // different attr order
							{Key: "dispute_id", Value: "1"},
						},
					},
					// Same type, different attrs — must be kept.
					{
						Type: "new_dispute",
						Attributes: []rpc.Attribute{
							{Key: "dispute_id", Value: "3"},
							{Key: "reporter", Value: "addrC"},
						},
					},
					// Unrelated event only in tx — kept.
					{
						Type: "deposit_claimed",
						Attributes: []rpc.Attribute{
							{Key: "deposit_id", Value: "9"},
						},
					},
				},
			},
		},
	}

	bv := BlockViewFromRPC(block)
	if len(bv.Events) != 4 {
		t.Fatalf("expected 4 events after cross-array dedupe, got %d: %+v", len(bv.Events), bv.Events)
	}

	var finalizeDisputes, txDisputes, deposits int
	for _, ev := range bv.Events {
		switch {
		case ev.Type == "new_dispute" && ev.Source == SourceFinalize:
			finalizeDisputes++
		case ev.Type == "new_dispute" && ev.Source == SourceTx:
			txDisputes++
			if ev.Attrs["dispute_id"] == "1" {
				t.Fatal("tx copy of dispute_id=1 should have been dropped")
			}
		case ev.Type == "deposit_claimed":
			deposits++
		}
	}
	if finalizeDisputes != 2 || txDisputes != 1 || deposits != 1 {
		t.Fatalf("finalizeDisputes=%d txDisputes=%d deposits=%d", finalizeDisputes, txDisputes, deposits)
	}
}
