package dispute

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	err := CheckOpenDisputesForExpiration(ctx, k)
	if err != nil {
		return err
	}
	return CheckClosedDisputesForExecution(ctx, k)
}

// SetBlockInfo logic should be in EndBlocker so that BlockInfo records the correct values after all delegations and tip additions for the block have been processed
func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	k.Logger(ctx).Info("IN ENDBLOCKER FOR DISPUTE")
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
			fmt.Println("SETTING BLOCK INFO AFTER DISPUTE OPENED IN END BLOCKER")
			k.Logger(ctx).Info("FOUND NEW OPEN DISPUTE AND SET BLOCK INFO")
			err := k.SetBlockInfo(ctx, dispute.HashId)
			if err != nil {
				return err
			}
			k.Logger(ctx).Info("FOUND NEW OPEN DISPUTE AND SET BLOCK INFO")
		}
	}
	return nil
}

// Checks for expired prevote disputes and sets them to failed if expired.
// Also checks whether any open disputes' vote periods have ended and tallies the vote if so.
func CheckOpenDisputesForExpiration(ctx context.Context, k keeper.Keeper) error {
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
			if sdk.UnwrapSDKContext(ctx).BlockTime().After(vote.VoteEnd) && vote.VoteResult == types.VoteResult_NO_TALLY {
				if err := k.TallyVote(ctx, key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

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
	return nil
}
