package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/mint/types"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Init(goCtx context.Context, msg *types.MsgInit) (*types.MsgMsgInitResponse, error) {
	if k.Keeper.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.Keeper.GetAuthority(), msg.Authority)
	}
	minter, err := k.Minter.Get(goCtx)
	if err != nil {
		return nil, err
	}
	if minter.Initialized {
		return nil, types.ErrAlreadyInitialized
	}
	minter.Initialized = true
	if err := k.Minter.Set(goCtx, minter); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"minter_initialized",
		),
	})
	return &types.MsgMsgInitResponse{}, nil
}

func (k msgServer) UpdateExtraRewardRate(ctx context.Context, msg *types.MsgUpdateExtraRewardRate) (*types.MsgUpdateExtraRewardRateResponse, error) {
	if k.Keeper.GetAuthority() != msg.Authority {
		return nil, errors.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.Keeper.GetAuthority(), msg.Authority)
	}
	extraRewardParams, err := k.Keeper.ExtraRewardParams.Get(ctx)
	if err != nil {
		return nil, err
	}
	if msg.DailyExtraRewards <= 0 {
		return nil, errors.Wrapf(types.ErrInvalidRequest, "daily extra rewards must be positive: %d", msg.DailyExtraRewards)
	}
	extraRewardParams.DailyExtraRewards = msg.DailyExtraRewards
	if err := k.Keeper.ExtraRewardParams.Set(ctx, extraRewardParams); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_extra_reward_rate",
			sdk.NewAttribute("daily_extra_rewards_rate", fmt.Sprintf("%d", msg.DailyExtraRewards)),
		),
	})
	return &types.MsgUpdateExtraRewardRateResponse{}, nil
}
