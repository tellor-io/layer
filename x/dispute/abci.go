package dispute

import (
	"context"
	"time"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyBeginBlocker)
	k.Logger(ctx).Info("Start time dispute module begin block: ", time.Now().UnixMilli())
	err := CheckOpenDisputesForExpiration(ctx, k)
	if err != nil {
		return err
	}
	return CheckClosedDisputesForExecution(ctx, k)
}

// SetBlockInfo logic should be in EndBlocker so that BlockInfo records the correct values after all delegations and tip additions for the block have been processed
func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
	k.Logger(ctx).Info("Start time dispute module end block: ", time.Now().UnixMilli())
	// check if a dispute has been opened at the current block height
	iter, err := k.Disputes.Indexes.OpenDisputes.MatchExact(ctx, true)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return err
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if dispute.BlockNumber == uint64(sdkCtx.BlockHeight()) {
			err := k.SetBlockInfo(ctx, dispute.HashId)
			if err != nil {
				return err
			}
			k.Logger(ctx).Info("FOUND NEW OPEN DISPUTE AND SET BLOCK INFO")
		}
	}
	k.Logger(ctx).Info("End time dispute module end block: ", time.Now().UnixMilli())
	return nil
}

// Checks for expired prevote disputes and sets them to failed if expired.
// Also checks whether any open disputes' vote periods have ended and tallies the vote if so.
func CheckOpenDisputesForExpiration(ctx context.Context, k keeper.Keeper) error {
	iter, err := k.Disputes.Indexes.OpenDisputes.MatchExact(ctx, true)
	if err != nil {
		return err
	}
	// do a 1000 open disputes at a time
	i := 1000
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		if i == 0 {
			break
		}
		key, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return err
		}
		// dispute is expired before it entered voting phase; so close dispute and set status to failed
		if sdk.UnwrapSDKContext(ctx).BlockTime().After(dispute.DisputeEndTime) && dispute.DisputeStatus == types.Prevote {
			dispute.Open = false
			dispute.DisputeStatus = types.Failed
			if err := k.Disputes.Set(ctx, key, dispute); err != nil {
				return err
			}
		} else if dispute.DisputeStatus == types.Voting {
			// try to tally the vote
			vote, err := k.Votes.Get(ctx, key)
			if err != nil {
				return err
			}
			// tally the vote if vote period ended and it hasn't been tallied yet
			if sdk.UnwrapSDKContext(ctx).BlockTime().After(vote.VoteEnd) && vote.VoteResult == types.VoteResult_NO_TALLY {
				if err := k.TallyVote(ctx, key); err != nil {
					return err
				}
			}
		}
		i--
	}
	return nil
}

// Checks if any disputes are pending execution, and if so, executes the vote.
func CheckClosedDisputesForExecution(ctx context.Context, k keeper.Keeper) error {
	iter, err := k.Disputes.Indexes.PendingExecution.MatchExact(ctx, true)
	if err != nil {
		return err
	}
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return err
		}
		if sdk.UnwrapSDKContext(ctx).BlockTime().After(dispute.DisputeEndTime) || dispute.DisputeStatus == types.Resolved {
			if err := k.ExecuteVote(ctx, key); err != nil {
				return err
			}
		}
	}
	k.Logger(ctx).Info("End time dispute module begin block: ", time.Now().UnixMilli())
	return nil
}
