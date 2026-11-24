package keeper

import (
	"context"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// WithdrawFeeRefund settles the named round-one payer after funding fails or a vote executes.
func (k msgServer) WithdrawFeeRefund(ctx context.Context, msg *types.MsgWithdrawFeeRefund) (*types.MsgWithdrawFeeRefundResponse, error) {
	// should be ok to be called by anyone
	feePayer, err := sdk.AccAddressFromBech32(msg.PayerAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid payer address (%s)", err)
	}
	// dispute
	dispute, err := k.Disputes.Get(ctx, msg.Id)
	if err != nil {
		return nil, err
	}

	payerKey := collections.Join(firstRoundID(msg.Id, dispute), feePayer.Bytes())
	payerInfo, err := k.DisputeFeePayer.Get(ctx, payerKey)
	if err != nil {
		return nil, err
	}
	remainder, err := k.Dust.Get(ctx)
	if err != nil {
		return nil, err
	}
	if dispute.DisputeStatus == types.Failed {
		if err := k.RefundFailedDisputeFee(ctx, feePayer, payerInfo, dispute.HashId); err != nil {
			return nil, err
		}
	} else {
		claim, err := k.resolvedPayerClaim(ctx, msg.Id, dispute, feePayer, payerInfo)
		if err != nil {
			return nil, err
		}
		if err := k.returnPayerFee(ctx, feePayer, payerInfo, dispute.HashId, claim.fee); err != nil {
			return nil, err
		}
		if err := k.returnPayerPrincipal(ctx, feePayer, claim.principal); err != nil {
			return nil, err
		}
		remainder = remainder.Add(claim.dust)
	}

	burnDust := remainder.Quo(layertypes.PowerReduction)
	if !burnDust.IsZero() {
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, burnDust))); err != nil {
			return nil, err
		}
		remainder = remainder.Mod(layertypes.PowerReduction)
	}

	if err := k.Dust.Set(ctx, remainder); err != nil {
		return nil, err
	}

	if err := k.DisputeFeePayer.Remove(ctx, payerKey); err != nil {
		return nil, err
	}
	return nil, nil
}
