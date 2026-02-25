package types

import (
	"testing"
)

// InitialCycleList uses hardcoded hex strings and panics on decode failure.
// This test verifies the hardcoded data is always valid.
func FuzzInitialCycleList(f *testing.F) {
	f.Add(0)
	f.Fuzz(func(t *testing.T, _ int) {
		// Should never panic
		result := InitialCycleList()
		if len(result) == 0 {
			t.Error("expected non-empty cycle list")
		}
		for i, item := range result {
			if len(item) == 0 {
				t.Errorf("cycle list item %d is empty", i)
			}
		}
	})
}
