package types

// StakeSnapshotRef points to the saved stake snapshot that backed a report.
type StakeSnapshotRef struct {
	// Block is the block the snapshot was saved in.
	Block uint64
	// Version is the snapshot's number; zero means the report was made before versions existed.
	Version uint64
}

// IsSet reports whether the reference points to a versioned snapshot.
func (r StakeSnapshotRef) IsSet() bool {
	return r.Version > 0
}
