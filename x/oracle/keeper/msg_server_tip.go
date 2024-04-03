package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (k msgServer) Tip(goCtx context.Context, msg *types.MsgTip) (*types.MsgTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}
	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)

	tip, err := k.Keeper.transfer(ctx, tipper, msg.Amount)
	if err != nil {
		return nil, err
	}

	// get query id bytes hash from query data
	queryId := utils.QueryIDFromData(msg.QueryData)

	// get query info for the query id
	query, err := k.Keeper.Query.Get(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		// initialize query tip first time
		query, err := k.Keeper.initializeQuery(ctx, msg.QueryData)
		if err != nil {
			return nil, err
		}

		query.Amount = tip.Amount
		query.Expiration = ctx.BlockTime().Add(query.RegistrySpecTimeframe)
		err = k.Keeper.Query.Set(ctx, queryId, query)
		if err != nil {
			return nil, err
		}
		return &types.MsgTipResponse{}, nil
	}
	prevAmt := query.Amount
	query.Amount = query.Amount.Add(tip.Amount)

	// expired submission window
	if query.Expiration.Before(ctx.BlockTime()) {
		// query expired, create new expiration time and new id
		query.Expiration = ctx.BlockTime().Add(query.RegistrySpecTimeframe)
		// in aggregate you set revealed reports to false after pay out
		// so if query either has reports or is paid out then new id should be generated
		if query.HasRevealedReports || prevAmt.IsZero() {
			id, err := k.QuerySequnecer.Next(ctx)
			if err != nil {
				return nil, err
			}
			query.Id = id
			query.Amount = tip.Amount
			query.HasRevealedReports = false
		}
	}
	err = k.Keeper.Query.Set(ctx, queryId, query)
	if err != nil {
		return nil, err
	}

	prevTip, err := k.Keeper.Tips.Get(ctx, collections.Join(queryId, tipper.Bytes()))
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, fmt.Errorf("failed to get previous tip: %w", err)
	}

	if !prevTip.IsNil() {
		tip = tip.AddAmount(prevTip)
	}
	err = k.Keeper.Tips.Set(ctx, collections.Join(queryId, tipper.Bytes()), tip.Amount)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.AddtoTotalTips(ctx, tip.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgTipResponse{}, nil
}
