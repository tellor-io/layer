package enrich_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/enrich"
)

func TestParseImportantReporters(t *testing.T) {
	got := enrich.ParseImportantReporters(" tellor1b,tellor1a, tellor1b , ,TELLOR1A ")
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %v", len(got), got)
	}
	// Sorted, first occurrence kept for casing.
	if got[0] != "tellor1a" && got[0] != "TELLOR1A" {
		t.Fatalf("got[0]=%q", got[0])
	}
	if enrich.ParseImportantReporters("") != nil {
		t.Fatal("empty should be nil")
	}
	if enrich.ParseImportantReporters("  , , ") != nil {
		t.Fatal("whitespace-only should be nil")
	}
}

func TestReporterMapDisplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reporters.json")
	content := `{
  "addressToMonikerMap": {
    "tellor1abc": "alpha",
    "Tellor1DEF": "beta"
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := enrich.NewReporterMap(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Display("tellor1abc"); got != "alpha" {
		t.Fatalf("Display abc = %q", got)
	}
	if got := m.Display("tellor1def"); got != "beta" {
		t.Fatalf("Display def = %q", got)
	}
	if got := m.Display("tellor1unknown"); got != "tellor1unknown" {
		t.Fatalf("unknown = %q", got)
	}
}

func TestFormatMissingReporters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reporters.json")
	content := `{"addressToMonikerMap":{"tellor1a":"Alice","tellor1b":"Bob"}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := enrich.NewReporterMap(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	important := []string{"tellor1a", "tellor1b", "tellor1c"}
	submitted := []string{"tellor1A"} // only Alice reported
	got := enrich.MissingReporters(important, submitted, m)
	if len(got) != 2 || got[0] != "Bob" || got[1] != "tellor1c" {
		t.Fatalf("MissingReporters = %v", got)
	}
	if enrich.FormatMissingReporters(important, submitted, m) != "Bob, tellor1c" {
		t.Fatalf("Format = %q", enrich.FormatMissingReporters(important, submitted, m))
	}
	if enrich.MissingReporters(important, []string{"tellor1a", "tellor1b", "tellor1c"}, m) != nil {
		t.Fatal("expected nil when none missing")
	}
	if enrich.MissingReporters(nil, submitted, m) != nil {
		t.Fatal("expected nil when no important")
	}
}
