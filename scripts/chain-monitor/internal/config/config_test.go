package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/config"
)

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := `
node_name: test-node
dry_run: true
rpc:
  urls:
    - "127.0.0.1:26657"
  timeout: 10s
  min_interval: 100ms
state:
  cursor_path: ./data/cursor.json
health:
  listen: ":9090"
start_from:
  tip: true
channels:
  bridge:
    webhook_url: ""
rules:
  - id: deposit_claimed
    channel: bridge
    match:
      event_type: deposit_claimed
    embed:
      title: "Bridge Deposit"
      color: 0x2ECC71
      fields:
        - { attr: deposit_id, name: "Deposit ID", format: code }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NodeName != "test-node" {
		t.Fatalf("NodeName = %q", cfg.NodeName)
	}
	if cfg.RPC.URLs[0] != "http://127.0.0.1:26657" {
		t.Fatalf("URL normalize = %q", cfg.RPC.URLs[0])
	}
	if cfg.RPC.Timeout != 10*time.Second {
		t.Fatalf("Timeout = %v", cfg.RPC.Timeout)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].ID != "deposit_claimed" {
		t.Fatalf("rules = %+v", cfg.Rules)
	}
	types := cfg.EventTypes()
	if _, ok := types["deposit_claimed"]; !ok {
		t.Fatal("expected deposit_claimed in event types")
	}
}

func TestValidateRequiresWebhookUnlessDryRun(t *testing.T) {
	cfg := &config.Config{
		NodeName:  "x",
		State:     config.StateConfig{CursorPath: "./c.json"},
		StartFrom: config.StartFromConfig{Tip: true},
		Channels:  map[string]config.Channel{"bridge": {}},
		Rules: []config.RuleConfig{{
			ID:      "r1",
			Channel: "bridge",
			Match:   config.MatchConfig{EventType: "deposit_claimed"},
			Embed:   config.EmbedConfig{Title: "T"},
		}},
	}
	cfg.RPC.URLs = []string{"http://127.0.0.1:26657"}
	cfg.RPC.Timeout = time.Second
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected webhook required error")
	}
	cfg.DryRun = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("dry_run should allow empty webhook: %v", err)
	}
}

func TestValidateBridgeDepositEnrichRequiresBridgeChannel(t *testing.T) {
	cfg := &config.Config{
		NodeName:  "x",
		DryRun:    true,
		State:     config.StateConfig{CursorPath: "./c.json"},
		StartFrom: config.StartFromConfig{Tip: true},
		Channels:  map[string]config.Channel{"oracle": {}},
		Rules: []config.RuleConfig{{
			ID:      "weak_aggregate",
			Channel: "oracle",
			Match:   config.MatchConfig{EventType: "aggregate_report"},
			Enrich:  []string{"bridge_deposit"},
			Embed:   config.EmbedConfig{Title: "T"},
		}},
	}
	cfg.RPC.URLs = []string{"http://127.0.0.1:26657"}
	cfg.RPC.Timeout = time.Second
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected bridge channel required error")
	}
	cfg.Channels["bridge"] = config.Channel{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate with bridge channel: %v", err)
	}
}

func TestValidateRequiresRPC(t *testing.T) {
	cfg := &config.Config{
		NodeName:  "x",
		State:     config.StateConfig{CursorPath: "./c.json"},
		StartFrom: config.StartFromConfig{Tip: true},
	}
	cfg.RPC.Timeout = time.Second
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty rpc.urls")
	}
}
