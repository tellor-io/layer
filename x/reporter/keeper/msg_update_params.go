package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/reporter/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	err := validateUpdateParams(req)
	if err != nil {
		return nil, err
	}
	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.Params.Set(ctx, req.Params); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"params_updated_by_authority",
			sdk.NewAttribute("max_selectors", strconv.FormatUint(req.Params.MaxSelectors, 10)),
			sdk.NewAttribute("min_loya", req.Params.MinLoya.String()),
			sdk.NewAttribute("min_commission_rate", req.Params.MinCommissionRate.String()),
		),
	})
	return &types.MsgUpdateParamsResponse{}, nil
}

func validateUpdateParams(m *types.MsgUpdateParams) error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	if err := m.Params.Validate(); err != nil {
		return err
	}

	return nil
}
