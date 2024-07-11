package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
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
	queries := make([]string, len(req.Cyclelist))
	for i, query := range req.Cyclelist {
		queries[i] = hex.EncodeToString(query)
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"cyclelist_updated",
			sdk.NewAttribute("cyclelist", fmt.Sprintf("%v", queries)),
		),
	})
	return &types.MsgUpdateCyclelistResponse{}, nil
}
