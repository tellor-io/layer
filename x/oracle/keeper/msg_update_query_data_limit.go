package keeper

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpdateQueryDataLimit updates the query data size limit.
// Gated function that can only be called by the x/gov.
// Emits a query_data_limit_updated event.
func (k msgServer) UpdateQueryDataLimit(ctx context.Context, req *types.MsgUpdateQueryDataLimit) (*types.MsgUpdateQueryDataLimitResponse, error) {
	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}

	if err := k.keeper.QueryDataLimit.Set(ctx, types.QueryDataLimit{Limit: req.Limit}); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"query_data_limit_updated",
			sdk.NewAttribute("limit", fmt.Sprintf("%v", req.Limit)),
		),
	})
	return &types.MsgUpdateQueryDataLimitResponse{}, nil
}
