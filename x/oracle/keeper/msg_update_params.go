package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"
)

// UpdateParams updates the oracle module's parameters.
// Gated function that can only be called by the x/gov.
// Note: Only param is the `MinStakeAmount`
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}

	if err := k.keeper.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
