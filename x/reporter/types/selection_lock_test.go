package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestSelectorStakeLocked(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("zero times", func(t *testing.T) {
		require.False(t, types.SelectorStakeLocked(types.Selection{}, now))
	})

	t.Run("locked_until only", func(t *testing.T) {
		sel := types.Selection{LockedUntilTime: now.Add(time.Hour)}
		require.True(t, types.SelectorStakeLocked(sel, now))
		sel.LockedUntilTime = now.Add(-time.Hour)
		require.False(t, types.SelectorStakeLocked(sel, now))
	})

	t.Run("dispute_locked_until only", func(t *testing.T) {
		sel := types.Selection{DisputeLockedUntil: now.Add(time.Hour)}
		require.True(t, types.SelectorStakeLocked(sel, now))
		sel.DisputeLockedUntil = now
		require.False(t, types.SelectorStakeLocked(sel, now))
	})

	t.Run("either field locks", func(t *testing.T) {
		sel := types.Selection{
			LockedUntilTime:    now.Add(-time.Hour),
			DisputeLockedUntil: now.Add(time.Hour),
		}
		require.True(t, types.SelectorStakeLocked(sel, now))
	})
}
