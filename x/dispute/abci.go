package dispute

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyBeginBlocker)
	err := CheckOpenDisputesForExpiration(ctx, k)
	if err != nil {
		return err
	}
	return CheckClosedDisputesForExecution(ctx, k)
}

// SetBlockInfo logic should be in EndBlocker so that BlockInfo records the correct values after all delegations and tip additions for the block have been processed
func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyEndBlocker)
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
	i := 1000
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := sdkCtx.BlockTime()
	processed := 0
	maxAge := time.Duration(0)
	for ; iter.Valid(); iter.Next() {
		if i == 0 {
			break
		}
		i--
		key, err := iter.PrimaryKey()
		if err != nil {
			return err
		}
		dispute, err := k.Disputes.Get(ctx, key)
		if err != nil {
			return err
		}
		if age := blockTime.Sub(dispute.DisputeEndTime); age > maxAge {
			maxAge = age
		}
		processed++
		if blockTime.After(dispute.DisputeEndTime) || dispute.DisputeStatus == types.Resolved {
			cacheCtx, writeCache := sdkCtx.CacheContext()
			if err := k.ExecuteVote(cacheCtx, key); err != nil {
				if errors.Is(err, reportertypes.ErrExceedsMaxValidatorPowerShare) ||
					errors.Is(err, stakingtypes.ErrDelegatorShareExRateInvalid) {
					sdkCtx.Logger().Info("dispute execution deferred", "dispute_id", key, "err", err)
					sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
						"dispute_execution_deferred",
						sdk.NewAttribute("dispute_id", strconv.FormatUint(key, 10)),
						sdk.NewAttribute("reason", err.Error()),
					))
					telemetry.IncrCounter(1, "dispute", "execution", "deferred")
					continue
				}
				sdkCtx.Logger().Error("dispute execution failed", "dispute_id", key, "err", err)
				sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
					"dispute_execution_failed",
					sdk.NewAttribute("dispute_id", strconv.FormatUint(key, 10)),
					sdk.NewAttribute("reason", err.Error()),
				))
				telemetry.IncrCounter(1, "dispute", "execution", "failed")
				continue
			}
			writeCache()
		}
	}
	if processed > 0 {
		telemetry.SetGauge(float32(processed), "dispute", "execution", "processed_count")
		telemetry.SetGauge(float32(maxAge.Seconds()), "dispute", "execution", "max_deferral_age_seconds")
	}
	return nil
}
