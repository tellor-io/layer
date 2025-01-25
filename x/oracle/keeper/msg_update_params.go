package keeper

import (
	"context"

	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpdateParams updates the oracle module's parameters.
// Gated function that can only be called by the x/gov.
// Note: Only param is the `MinStakeAmount`
func (k msgServer) UpdateParams(ctx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if err := validateUpdateParams(req); err != nil {
		return nil, err
	}

	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}

	if err := k.keeper.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func validateUpdateParams(msg *types.MsgUpdateParams) error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := msg.Params.Validate(); err != nil {
		return err
	}

	return nil
}
