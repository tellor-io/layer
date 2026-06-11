package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
)

// GetMaxOpenCommitmentForReporter returns the monotonic max query Expiration
// height seen for reports submitted by this reporter (bumped on each MsgSubmitValue).
// Missing entry means 0. The reporter module reads this when scheduling a pending
// switch so unlock_block reflects the outgoing reporter's committed query windows.
func (k Keeper) GetMaxOpenCommitmentForReporter(ctx context.Context, reporter []byte) (uint64, error) {
	v, err := k.MaxOpenCommitmentByReporter.Get(ctx, reporter)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// bumpMaxOpenCommitmentForReporter raises the stored max when queryExpiration is higher.
func (k Keeper) bumpMaxOpenCommitmentForReporter(ctx context.Context, reporter []byte, queryExpiration uint64) error {
	cur, err := k.MaxOpenCommitmentByReporter.Get(ctx, reporter)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	}
	if errors.Is(err, collections.ErrNotFound) || queryExpiration > cur {
		return k.MaxOpenCommitmentByReporter.Set(ctx, reporter, queryExpiration)
	}
	return nil
}
