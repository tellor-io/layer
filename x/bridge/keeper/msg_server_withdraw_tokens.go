package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) WithdrawTokens(goCtx context.Context, msg *types.MsgWithdrawTokens) (*types.MsgWithdrawTokensResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	fmt.Println("denom: ", msg.Amount.Denom)
	fmt.Println("amount: ", msg.Amount.Amount)
	fmt.Println("isZero: ", msg.Amount.Amount.IsZero())
	fmt.Println("isNegative: ", msg.Amount.Amount.IsNegative())
	fmt.Println()
	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}

	sender := sdk.MustAccAddressFromBech32(msg.Creator)
	fmt.Println("msg.Creator: ", msg.Creator)
	fmt.Println("sender: ", sender)

	recipient, err := hex.DecodeString(msg.Recipient)
	fmt.Println("msg.Recipient: ", msg.Recipient)
	fmt.Println("recipient: ", recipient)
	fmt.Println("err: ", err)

	if err != nil {
		return nil, sdkerrors.ErrInvalidRequest
	}

	if err := k.withdrawTokens(sdkCtx, msg.Amount, sender, recipient); err != nil {
		return nil, err
	}
	return &types.MsgWithdrawTokensResponse{}, nil
}
