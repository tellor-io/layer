package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/state"
)

func TestValsetAppendAndRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "valset.json")
	store := state.NewValsetStore(path)

	now := time.Now().UTC()
	if err := store.Append(now.Add(-48 * time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(now.Add(-24 * time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	recent, err := store.Recent(36 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent=%d want 2", len(recent))
	}

	count, avg, median, latest := state.AnalyzeFrequency(recent)
	if count != 2 || latest.IsZero() || avg <= 0 || median <= 0 {
		t.Fatalf("analysis count=%d avg=%v median=%v latest=%v", count, avg, median, latest)
	}
}
