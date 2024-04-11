package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

func (k msgServer) WithdrawTokens(goCtx context.Context, msg *types.MsgWithdrawTokens) (*types.MsgWithdrawTokensResponse, error) {
	k.Logger(goCtx).Info("@WithdrawTokens msgserver", "creator", msg.Creator, "recipient", msg.Recipient, "amount", msg.Amount)
	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}
	sender := sdk.MustAccAddressFromBech32(msg.Creator)
	recipient, err := hex.DecodeString(msg.Recipient)
	if err != nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if err := k.withdrawTokens(sdkCtx, msg.Amount, sender, recipient); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawTokensResponse{}, nil
}
