package types

import "time"

// SelectorStakeLocked returns true when selector stake must be excluded from reporting power.
func SelectorStakeLocked(sel Selection, now time.Time) bool {
	return sel.LockedUntilTime.After(now) || sel.DisputeLockedUntil.After(now)
}
