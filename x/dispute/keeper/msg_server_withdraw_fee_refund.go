package keeper

import (
	"context"
	"errors"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) WithdrawFeeRefund(ctx context.Context, msg *types.MsgWithdrawFeeRefund) (*types.MsgWithdrawFeeRefundResponse, error) {
	k.Keeper.Logger(ctx).Info("WithdrawFeeRefund")
	// should be ok to be called by anyone
	feePayer := sdk.MustAccAddressFromBech32(msg.PayerAddress)
	// dispute
	dispute, err := k.Disputes.Get(ctx, msg.Id)
	if err != nil {
		return nil, err
	}

	// get previous disputes
	prevDisputes := dispute.PrevDisputeIds
	var firstRoundDisputeId uint64
	if len(prevDisputes) == 0 {
		firstRoundDisputeId = msg.Id
	} else {
		firstRoundDisputeId = prevDisputes[0]
	}
	// get first round that to
	// rd1Dispute, err := k.Disputes.Get(ctx, firstRoundDisputeId)
	if err != nil {
		return nil, err
	}
	// get dispute fee payer from rd 1
	payerInfo, err := k.DisputeFeePayer.Get(ctx, collections.Join(firstRoundDisputeId, feePayer.Bytes()))
	if err != nil {
		return nil, err
	}
	remainder, err := k.Dust.Get(ctx)
	if err != nil {
		return nil, err
	}
	// handle failed underfunded dispute
	if dispute.DisputeStatus == types.Failed {
		err := k.RefundFailedDisputeFee(ctx, feePayer, payerInfo, dispute.HashId)
		if err != nil {
			return nil, err
		}

	} else {
		// check if vote executed
		vote, err := k.Votes.Get(ctx, msg.Id)
		if err != nil {
			return nil, err
		}

		if !vote.Executed {
			return nil, errors.New("vote not executed")
		}

		switch vote.VoteResult {
		case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
			fraction, err := k.RefundDisputeFee(ctx, feePayer, payerInfo, dispute.FeeTotal, dispute.HashId, dispute.SlashAmount)
			if err != nil {
				return nil, err
			}
			remainder = remainder.Add(fraction)
		case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
			fraction, err := k.RefundDisputeFee(ctx, feePayer, payerInfo, dispute.FeeTotal, dispute.HashId, dispute.SlashAmount)
			if err != nil {
				return nil, err
			}

			remainder = remainder.Add(fraction)

			fraction, err = k.RewardReporterBondToFeePayers(ctx, feePayer, payerInfo, dispute.FeeTotal, dispute.SlashAmount)
			if err != nil {
				return nil, err
			}

			remainder = remainder.Add(fraction)

		default:
			return nil, errors.New("invalid vote result")
		}

	}

	remainderDec := math.LegacyNewDecFromInt(remainder)
	powerReductionDec := math.LegacyNewDecFromInt(layertypes.PowerReduction)
	burnDustDec := remainderDec.Quo(powerReductionDec)
	burnDust := burnDustDec.TruncateInt()

	if !burnDust.IsZero() {
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burnDust))); err != nil {
			return nil, err
		}
		remainder = remainder.Mod(layertypes.PowerReduction)
	}

	if err := k.Dust.Set(ctx, remainder); err != nil {
		return nil, err
	}

	if err := k.DisputeFeePayer.Remove(ctx, collections.Join(firstRoundDisputeId, feePayer.Bytes())); err != nil {
		return nil, err
	}
	return nil, nil
}
