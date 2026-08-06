package metrics

import (
	"strings"
	"testing"
)

func TestExposeBasics(t *testing.T) {
	r := New()
	r.BlocksProcessed.Add(3)
	r.RPCErrors.Add(1)
	r.IncAlertSent("weak_aggregate")
	r.IncAlertSent("weak_aggregate")
	r.IncAlertSent("new_dispute")
	r.AlertsDeduped.Add(4)
	r.AlertsRateLimited.Add(2)
	r.SetHeights(100, 105)
	r.SetValidatorPower(9001, true)

	out := r.Expose()
	checks := []string{
		"chain_monitor_blocks_processed_total 3",
		"chain_monitor_rpc_errors_total 1",
		"chain_monitor_alerts_sent_total 3",
		`chain_monitor_alerts_sent_by_rule_total{rule="new_dispute"} 1`,
		`chain_monitor_alerts_sent_by_rule_total{rule="weak_aggregate"} 2`,
		"chain_monitor_cursor_height 100",
		"chain_monitor_tip_height 105",
		"chain_monitor_ingest_lag 5",
		"chain_monitor_validator_power 9001",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestValidatorPowerUnknown(t *testing.T) {
	r := New()
	r.SetValidatorPower(123, false)
	out := r.Expose()
	if !strings.Contains(out, "chain_monitor_validator_power 0") {
		t.Fatalf("expected 0 when unknown:\n%s", out)
	}
}
