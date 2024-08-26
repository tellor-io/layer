package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimDeposit(goCtx context.Context, msg *types.MsgClaimDepositRequest) (*types.MsgClaimDepositResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	msgSender := sdk.MustAccAddressFromBech32(msg.Creator)
	if err := k.Keeper.ClaimDeposit(sdkCtx, msg.DepositId, msg.Index, msgSender); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"deposit_claimed",
			sdk.NewAttribute("deposit_id", strconv.FormatUint(msg.DepositId, 10)),
		),
	})

	return &types.MsgClaimDepositResponse{}, nil
}
