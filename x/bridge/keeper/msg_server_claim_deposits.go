package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/bridge/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Claim deposits made by the Ethereum token bridge contract into Layer.
func (k msgServer) ClaimDeposits(goCtx context.Context, msg *types.MsgClaimDepositsRequest) (*types.MsgClaimDepositsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	if len(msg.DepositIds) != len(msg.Timestamps) {
		return nil, types.ErrInvalidDepositIdsAndIndicesLength
	}
	for i, depositId := range msg.DepositIds {
		timestamp := msg.Timestamps[i]
		if err := k.Keeper.ClaimDeposit(sdkCtx, depositId, timestamp); err != nil {
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
