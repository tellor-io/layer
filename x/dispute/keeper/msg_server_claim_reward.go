package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) ClaimReward(ctx context.Context, msg *types.MsgClaimReward) (*types.MsgClaimRewardResponse, error) {
	callerAcc, err := sdk.AccAddressFromBech32(msg.CallerAddress)
	k.Logger(ctx).Info("ClaimReward caller acc", "ClaimReward caller acc", callerAcc)
	if err != nil {
		return nil, err
	}
	cosmosCtx := sdk.UnwrapSDKContext(ctx)
	err = k.Keeper.ClaimReward(cosmosCtx, callerAcc, msg.DisputeId)
	if err != nil {
		k.Logger(ctx).Error("ClaimReward error", "ClaimReward", err)
		return nil, err
	}

	return &types.MsgClaimRewardResponse{}, nil
}
