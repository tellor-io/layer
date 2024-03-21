package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (k msgServer) UpdateCyclelist(goCtx context.Context, req *types.MsgUpdateCyclelist) (*types.MsgUpdateCyclelistResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.Cyclelist.Clear(ctx, nil); err != nil {
		return nil, err
	}
	if err := k.InitCycleListQuery(ctx, req.Cyclelist); err != nil {
		return nil, err
	}

	return &types.MsgUpdateCyclelistResponse{}, nil
}
