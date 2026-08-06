package rules_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/power"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rpc"
	"github.com/tellor-io/layer/scripts/chain-monitor/internal/rules"
)

type fixtureFile struct {
	Name           string              `yaml:"name"`
	Description    string              `yaml:"description"`
	ValidatorPower uint64              `yaml:"validator_power"`
	PowerKnown     *bool               `yaml:"power_known"`
	Block          fixtureBlock        `yaml:"block"`
	Rules          []config.RuleConfig `yaml:"rules"`
	Expect         fixtureExpect       `yaml:"expect"`
}

type fixtureBlock struct {
	Height  uint64         `yaml:"height"`
	Time    string         `yaml:"time"`
	ChainID string         `yaml:"chain_id"`
	Events  []fixtureEvent `yaml:"events"`
}

type fixtureEvent struct {
	Type   string            `yaml:"type"`
	Source string            `yaml:"source"`
	Attrs  map[string]string `yaml:"attrs"`
}

type fixtureExpect struct {
	AlertCount int                  `yaml:"alert_count"`
	Alerts     []fixtureExpectAlert `yaml:"alerts"`
}

type fixtureExpectAlert struct {
	RuleID  string            `yaml:"rule_id"`
	Channel string            `yaml:"channel"`
	Title   string            `yaml:"title"`
	Fields  map[string]string `yaml:"fields"`
}

type staticPower struct{ p uint64 }

func (s staticPower) TotalVotingPower(context.Context) (uint64, error) { return s.p, nil }

func TestFixtures(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "fixtures")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no fixtures found")
	}

	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".yml" {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		t.Run(ent.Name(), func(t *testing.T) {
			runFixture(t, path)
		})
	}
}

func runFixture(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx fixtureFile
	if err := yaml.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	channels := map[string]config.Channel{}
	for _, r := range fx.Rules {
		channels[r.Channel] = config.Channel{WebhookURL: "https://example.invalid/webhook"}
	}
	cfg := &config.Config{
		NodeName: "fixture-node",
		Channels: channels,
		Rules:    fx.Rules,
		DryRun:   true,
	}
	cfg.Defaults.RateLimit.Max = 100
	cfg.Defaults.RateLimit.Window = time.Hour
	cfg.Defaults.RateLimit.Cooldown = time.Hour

	var cache *power.Cache
	powerKnown := fx.ValidatorPower > 0
	if fx.PowerKnown != nil {
		powerKnown = *fx.PowerKnown
	}
	if powerKnown && fx.ValidatorPower > 0 {
		cache = power.NewCache(staticPower{p: fx.ValidatorPower}, time.Hour, nil, nil)
		if err := cache.Refresh(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	engine := rules.NewEngine(cfg, nil, cache, nil)
	block := toBlockView(t, fx.Block)
	alerts := engine.Evaluate(block)

	if len(alerts) != fx.Expect.AlertCount {
		t.Fatalf("%s: alert_count=%d want %d", fx.Name, len(alerts), fx.Expect.AlertCount)
	}
	for i, want := range fx.Expect.Alerts {
		if i >= len(alerts) {
			t.Fatalf("missing alert[%d]", i)
		}
		got := alerts[i]
		if got.RuleID != want.RuleID {
			t.Errorf("alert[%d].rule_id=%q want %q", i, got.RuleID, want.RuleID)
		}
		if got.Channel != want.Channel {
			t.Errorf("alert[%d].channel=%q want %q", i, got.Channel, want.Channel)
		}
		if got.Embed.Title != want.Title {
			t.Errorf("alert[%d].title=%q want %q", i, got.Embed.Title, want.Title)
		}
		gotFields := map[string]string{}
		for _, f := range got.Embed.Fields {
			gotFields[f.Name] = f.Value
		}
		for name, val := range want.Fields {
			if gotFields[name] != val {
				t.Errorf("alert[%d] field %q=%q want %q", i, name, gotFields[name], val)
			}
		}
	}
}

func toBlockView(t *testing.T, b fixtureBlock) rules.BlockView {
	t.Helper()
	bv := rules.BlockView{
		Height:  b.Height,
		ChainID: b.ChainID,
	}
	if b.Time != "" {
		ts, err := time.Parse(time.RFC3339, b.Time)
		if err != nil {
			t.Fatalf("parse block time: %v", err)
		}
		bv.Time = ts
	}
	for _, ev := range b.Events {
		src := rules.SourceFinalize
		if ev.Source == "tx" {
			src = rules.SourceTx
		}
		bv.Events = append(bv.Events, rules.EventView{
			Type:    ev.Type,
			Attrs:   ev.Attrs,
			Source:  src,
			TxIndex: -1,
		})
	}
	return bv
}

func TestBlockResultsFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "block_results_deposit.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Height              uint64         `json:"height"`
		Time                string         `json:"time"`
		ChainID             string         `json:"chain_id"`
		FinalizeBlockEvents []rpc.Event    `json:"finalize_block_events"`
		TxsResults          []rpc.TxResult `json:"txs_results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	block := &rpc.BlockResult{
		Height:              payload.Height,
		Time:                payload.Time,
		ChainID:             payload.ChainID,
		FinalizeBlockEvents: payload.FinalizeBlockEvents,
		TxsResults:          payload.TxsResults,
	}
	view := rules.BlockViewFromRPC(block)
	if view.Height != 25669427 {
		t.Fatalf("height=%d", view.Height)
	}
	if len(view.Events) != 1 || view.Events[0].Type != "deposit_claimed" {
		t.Fatalf("events=%+v", view.Events)
	}
	if view.Events[0].Attrs["amount"] != "5092440000loya" {
		t.Fatalf("attrs=%v", view.Events[0].Attrs)
	}
}

func TestSignalRules(t *testing.T) {
	cfg := &config.Config{
		NodeName: "n",
		Channels: map[string]config.Channel{
			"infra": {WebhookURL: "http://x"},
		},
		Rules: []config.RuleConfig{
			{
				ID:      "slow_blocks",
				Channel: "infra",
				Match:   config.MatchConfig{Kind: config.KindBlockInterval},
				When:    config.WhenConfig{MaxInterval: time.Minute},
				Embed: config.EmbedConfig{
					Title:  "Slow Blocks",
					Fields: []config.FieldConfig{{Attr: "normalized_interval", Name: "Interval"}},
				},
			},
			{
				ID:      "rpc_unhealthy",
				Channel: "infra",
				Match:   config.MatchConfig{Kind: config.KindRPCUnhealthy},
				When:    config.WhenConfig{FailThreshold: 3},
				Embed: config.EmbedConfig{
					Title:  "RPC Failures",
					Fields: []config.FieldConfig{{Attr: "consecutive_fails", Name: "Fails", Format: "code"}},
				},
			},
			{
				ID:      "ingest_lag",
				Channel: "infra",
				Match:   config.MatchConfig{Kind: config.KindIngestLag},
				When:    config.WhenConfig{MaxLag: 50},
				Embed: config.EmbedConfig{
					Title:  "Ingest Lag",
					Fields: []config.FieldConfig{{Attr: "lag", Name: "Lag", Format: "code"}},
				},
			},
		},
	}
	engine := rules.NewEngine(cfg, nil, nil, nil)

	t0 := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	if got := engine.EvaluateBlockInterval(10, 11, t0, t0.Add(2*time.Minute)); len(got) != 1 {
		t.Fatalf("slow blocks: got %d", len(got))
	}
	if got := engine.EvaluateBlockInterval(11, 12, t0, t0.Add(30*time.Second)); len(got) != 0 {
		t.Fatalf("fast blocks: got %d", len(got))
	}

	if got := engine.EvaluateRPCUnhealthy(2, "timeout"); len(got) != 0 {
		t.Fatalf("rpc below threshold: got %d", len(got))
	}
	if got := engine.EvaluateRPCUnhealthy(3, "timeout"); len(got) != 1 {
		t.Fatalf("rpc at threshold: got %d", len(got))
	}

	if got := engine.EvaluateIngestLag(100, 140); len(got) != 0 {
		t.Fatalf("lag under max: got %d", len(got))
	}
	if got := engine.EvaluateIngestLag(100, 160); len(got) != 1 {
		t.Fatalf("lag over max: got %d", len(got))
	}
}
