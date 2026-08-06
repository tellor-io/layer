package state_test

import (
	"path/filepath"
	"testing"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/state"
)

func TestCursorSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "cursor.json")
	c := state.NewCursorFile(path)

	_, exists, err := c.Load()
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected no cursor")
	}

	if err := c.Save(42); err != nil {
		t.Fatal(err)
	}

	h, exists, err := c.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !exists || h != 42 {
		t.Fatalf("got height=%d exists=%v", h, exists)
	}

	if err := c.Save(43); err != nil {
		t.Fatal(err)
	}
	h, _, err = c.Load()
	if err != nil || h != 43 {
		t.Fatalf("got height=%d err=%v", h, err)
	}
}
