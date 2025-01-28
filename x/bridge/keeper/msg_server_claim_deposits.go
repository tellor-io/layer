package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimDeposits(goCtx context.Context, msg *types.MsgClaimDepositsRequest) (*types.MsgClaimDepositsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	if len(msg.DepositIds) != len(msg.Timestamps) {
		return nil, types.ErrInvalidDepositIdsAndIndicesLength
	}
	msgSender := sdk.MustAccAddressFromBech32(msg.Creator)
	for i, depositId := range msg.DepositIds {
		timestamp := msg.Timestamps[i]
		if err := k.Keeper.ClaimDeposit(sdkCtx, depositId, timestamp, msgSender); err != nil {
			return nil, err
		}
		sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"deposit_claimed",
				sdk.NewAttribute("deposit_id", strconv.FormatUint(depositId, 10)),
			),
		})
	}

	return &types.MsgClaimDepositsResponse{}, nil
}
