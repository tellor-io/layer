package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"layer/x/oracle/types"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubmitValueResponse{}, nil
}
