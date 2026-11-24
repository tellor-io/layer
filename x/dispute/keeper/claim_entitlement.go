package keeper

import (
	"context"
	"errors"

	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	errVoteNotExecuted = errors.New("vote not executed")
	errInvalidResult   = errors.New("invalid vote result")
)

type payerClaim struct {
	fee       math.Int
	principal math.Int
	dust      math.Int // millionths of a loya from the two separately floored legs
}

func firstRoundID(id uint64, dispute types.Dispute) uint64 {
	if len(dispute.PrevDisputeIds) > 0 {
		return dispute.PrevDisputeIds[0]
	}
	return id
}

// resolvedPayerClaim is the source of both the query amount and the withdrawal.
func (k Keeper) resolvedPayerClaim(ctx context.Context, id uint64, dispute types.Dispute, payer sdk.AccAddress, info types.PayerInfo) (payerClaim, error) {
	claim := payerClaim{fee: math.ZeroInt(), principal: math.ZeroInt(), dust: math.ZeroInt()}
	vote, err := k.Votes.Get(ctx, id)
	if errors.Is(err, collections.ErrNotFound) {
		return claim, errVoteNotExecuted
	}
	if err != nil {
		return claim, err
	}
	if !vote.Executed {
		return claim, errVoteNotExecuted
	}

	quote, retained := dispute.SlashAmount, dispute.DisputeFee
	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		claim.fee, claim.dust = CalculateRefundAmount(info.Amount, quote, retained)
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		// Only SUPPORT retains the reporter's principal snapshot after execution.
		collected, err := k.reporterKeeper.DisputedDelegationTotal(ctx, dispute.HashId)
		if err != nil {
			return claim, err
		}
		claim.fee, claim.dust = CalculateRefundAmount(info.Amount, quote, retained)
		var principalDust math.Int
		claim.principal, principalDust = CalculateReporterBondRewardAmount(info.Amount, quote, collected)
		claim.dust = claim.dust.Add(principalDust)
	case types.VoteResult_AGAINST, types.VoteResult_NO_QUORUM_MAJORITY_AGAINST:
		if retained.GTE(quote) {
			return claim, errInvalidResult
		}
		claim.fee, claim.dust = CalculateAgainstRefundAmount(info.Amount, quote, retained)
	default:
		return claim, errInvalidResult
	}

	claim.fee, err = k.backedFeeRefund(ctx, dispute.HashId, payer, info, claim.fee)
	return claim, err
}

func (k Keeper) backedFeeRefund(ctx context.Context, hashID []byte, payer sdk.AccAddress, info types.PayerInfo, refund math.Int) (math.Int, error) {
	if !info.FromBond || info.Amount.IsZero() {
		return refund, nil
	}
	actual, err := k.reporterKeeper.FeePaidFromStakeTotalByPayer(ctx, hashID, payer)
	if err != nil {
		return math.Int{}, err
	}
	return CalculateActualStakeFeeRefundAmount(actual, info.Amount, refund), nil
}
