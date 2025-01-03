package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimReward(ctx context.Context, msg *types.MsgClaimReward) (*types.MsgClaimRewardResponse, error) {
	callerAcc, err := sdk.AccAddressFromBech32(msg.CallerAddress)
	if err != nil {
		return nil, err
	}
	cosmosCtx := sdk.UnwrapSDKContext(ctx)
	err = k.Keeper.ClaimReward(cosmosCtx, callerAcc, msg.DisputeId)
	if err != nil {
		return nil, err
	}

	return &types.MsgClaimRewardResponse{}, nil
}
