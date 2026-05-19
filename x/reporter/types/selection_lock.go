package types

import "time"

// SelectorStakeLocked returns true when selector stake must be excluded from reporting power.
func SelectorStakeLocked(sel Selection, now time.Time) bool {
	if sel.LockedUntilTime.After(now) {
		return true
	}
	return sel.Jailed && now.Before(sel.JailedUntil)
}
