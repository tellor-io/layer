package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/tellor-io/layer/x/oracle/types"
)

func (k msgServer) UpdateCyclelist(ctx context.Context, req *types.MsgUpdateCyclelist) (*types.MsgUpdateCyclelistResponse, error) {
	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}

	if err := k.keeper.Cyclelist.Clear(ctx, nil); err != nil {
		return nil, err
	}
	if err := k.keeper.InitCycleListQuery(ctx, req.Cyclelist); err != nil {
		return nil, err
	}

	return &types.MsgUpdateCyclelistResponse{}, nil
}
