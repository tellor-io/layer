package keeper

import (
	"context"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tellor-io/layer/x/registry/types"
)

func (k msgServer) UpdateDataSpec(goCtx context.Context, req *types.MsgUpdateDataSpec) (*types.MsgUpdateDataSpecResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	// normalize query type
	req.QueryType = strings.ToLower(req.QueryType)
	// check if the query type exists
	querytypeExists, err := k.Keeper.HasSpec(ctx, req.QueryType)
	if err != nil {
		return nil, err
	}
	if !querytypeExists {
		return nil, errorsmod.Wrapf(types.ErrInvalidSpec, "data spec not registered for query type: %s", req.QueryType)
	}
	if err := k.Keeper.SetDataSpec(ctx, req.QueryType, req.Spec); err != nil {
		return nil, err
	}

	if err := k.Keeper.Hooks().AfterDataSpecUpdated(ctx, req.QueryType, req.Spec); err != nil {
		return nil, err
	}

	return &types.MsgUpdateDataSpecResponse{}, nil
}
