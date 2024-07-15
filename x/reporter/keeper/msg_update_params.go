package keeper

import (
	"context"
	"strconv"

	"github.com/tellor-io/layer/x/reporter/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
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
			sdk.NewAttribute("min_trb", req.Params.MinTrb.String()),
			sdk.NewAttribute("min_commission_rate", req.Params.MinCommissionRate.String()),
		),
	})
	return &types.MsgUpdateParamsResponse{}, nil
}
