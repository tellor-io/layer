package keeper

import (
	"context"
	"strings"

	"github.com/tellor-io/layer/x/registry/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RemoveDataSpecs(goCtx context.Context, req *types.MsgRemoveDataSpecs) (*types.MsgRemoveDataSpecsResponse, error) {
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	for i := 0; i < len(req.DataSpecTypes); i++ {
		specType := strings.ToLower(req.DataSpecTypes[i])
		hasSpec, err := k.Keeper.HasSpec(ctx, specType)
		if err != nil {
			return &types.MsgRemoveDataSpecsResponse{}, err
		}
		if !hasSpec {
			return &types.MsgRemoveDataSpecsResponse{}, errorsmod.Wrapf(types.ErrSpecDoesNotExist, "%s was not found", specType)
		}

		err = k.Keeper.SpecRegistry.Remove(ctx, specType)
		if err != nil {
			return &types.MsgRemoveDataSpecsResponse{}, errorsmod.Wrapf(err, "Could not remove %s dataspec: %v", specType, err)
		}
	}

	return &types.MsgRemoveDataSpecsResponse{}, nil
}
