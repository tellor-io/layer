package keeper

import (
	"context"
	"errors"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) WithdrawFeeRefund(ctx context.Context, msg *types.MsgWithdrawFeeRefund) (*types.MsgWithdrawFeeRefundResponse, error) {
	// should be ok to be called by anyone
	feePayer := sdk.MustAccAddressFromBech32(msg.PayerAddress)
	// check if vote executed
	vote, err := k.Votes.Get(ctx, msg.Id)
	if err != nil {
		return nil, err
	}
	if !vote.Executed {
		return nil, errors.New("vote not executed")
	}

	payerInfo, err := k.DisputeFeePayer.Get(ctx, collections.Join(msg.Id, feePayer.Bytes()))
	if err != nil {
		return nil, err
	}

	// dispute
	dispute, err := k.Disputes.Get(ctx, msg.Id)
	if err != nil {
		return nil, err
	}

	feeMinusBurn := dispute.SlashAmount.Sub(dispute.BurnAmount)
	remainder, err := k.Dust.Get(ctx)
	if err != nil {
		return nil, err
	}

	switch vote.VoteResult {
	case types.VoteResult_INVALID, types.VoteResult_NO_QUORUM_MAJORITY_INVALID:
		fraction, err := k.RefundDisputeFee(ctx, feePayer, payerInfo, dispute.FeeTotal, feeMinusBurn, dispute.HashId)
		if err != nil {
			return nil, err
		}
		remainder = remainder.Add(fraction)
	case types.VoteResult_SUPPORT, types.VoteResult_NO_QUORUM_MAJORITY_SUPPORT:
		fraction, err := k.RefundDisputeFee(ctx, feePayer, payerInfo, dispute.FeeTotal, feeMinusBurn, dispute.HashId)
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

	burnDust := remainder.TruncateInt()

	if !burnDust.IsZero() {
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, burnDust))); err != nil {
			return nil, err
		}
		remainder = remainder.Sub(remainder.TruncateDec())
	}

	if err := k.Dust.Set(ctx, remainder); err != nil {
		return nil, err
	}

	if err := k.DisputeFeePayer.Remove(ctx, collections.Join(msg.Id, feePayer.Bytes())); err != nil {
		return nil, err
	}
	return nil, nil
}
