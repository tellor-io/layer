package keeper

import (
	"context"

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
	// currentTime := sdk.UnwrapSDKContext(goCtx).BlockTime()
	// minter.PreviousBlockTime = &currentTime
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
