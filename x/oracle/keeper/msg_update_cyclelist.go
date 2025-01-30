package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	regTypes "github.com/tellor-io/layer/x/registry/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpdateCyclelist updates the cyclelist with the provided list of queryData.
// Gated function that can only be called by the x/gov.
// Deletes entire current cyclelist queries and initializes the new cyclelist queries.
// Emits a cyclelist_updated event.
func (k msgServer) UpdateCyclelist(ctx context.Context, req *types.MsgUpdateCyclelist) (*types.MsgUpdateCyclelistResponse, error) {
	if k.keeper.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.keeper.GetAuthority(), req.Authority)
	}

	if len(req.Cyclelist) == 0 {
		return nil, errorsmod.Wrapf(fmt.Errorf("cyclelist is empty"), "cyclelist cannot be an empty array")
	}

	queries := make([]string, len(req.Cyclelist))
	for i, querydata := range req.Cyclelist {
		// decode the queryType
		queryType, fieldBytes, err := regTypes.DecodeQueryType(querydata)
		if err != nil {
			return nil, err
		}
		// check if the queryType is registered
		dataSpec, err := k.keeper.GetDataSpec(ctx, queryType)
		if err != nil {
			return nil, err
		}
		// check if the fieldBytes are valid for the queryType
		_, err = regTypes.DecodeParamtypes(fieldBytes, dataSpec.AbiComponents)
		if err != nil {
			return nil, err
		}
		queries[i] = hex.EncodeToString(querydata)
	}

	// if we make it here then the cyclelist is valid and will be updated
	if err := k.keeper.Cyclelist.Clear(ctx, nil); err != nil {
		return nil, err
	}
	if err := k.keeper.InitCycleListQuery(ctx, req.Cyclelist); err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"cyclelist_updated",
			sdk.NewAttribute("cyclelist", fmt.Sprintf("%v", queries)),
		),
	})
	return &types.MsgUpdateCyclelistResponse{}, nil
}
