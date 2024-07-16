package keeper

import (
	"context"
	"encoding/hex"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) WithdrawTokens(goCtx context.Context, msg *types.MsgWithdrawTokens) (*types.MsgWithdrawTokensResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}

	sender := sdk.MustAccAddressFromBech32(msg.Creator)

	recipient, err := hex.DecodeString(msg.Recipient)
	if err != nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if err := k.Keeper.WithdrawTokens(sdkCtx, msg.Amount, sender, recipient); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tokens_withdrawn",
			sdk.NewAttribute("sender", msg.Creator),
			sdk.NewAttribute("recipient_evm_address", msg.Recipient),
			sdk.NewAttribute("amount", msg.Amount.String()),
		),
	})
	return &types.MsgWithdrawTokensResponse{}, nil
}
