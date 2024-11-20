package keeper

import (
	"context"
	"encoding/hex"
	"errors"

	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Tip handles tipping a query; accepts query data and amount to tip.
// 1. Checks if the bond denom is correct and if the amount is positive.
// 2. Transfers the amount to the module account after burning 2% of the tip.
// 3. Fetches the QueryMeta by queryId:
//   - If QueryMeta is not found, initializes a new QueryMeta and sets the amount and the expiration time.
//   - If QueryMeta is found the tip in increased by the new tip amount. Then the expiration time is checked
//     to see if the query is expired. If the query is expired, the expiration is extended according to the registry spec otherwise do nothing.
//
// 4. Add the tip amount to the tipper's total and the total tips.
// Note:
//
//	If a query has expired, and the prev.Amount is not zero, then that means the query has no reports. If it has entered this current block
//	that means the query is expired and no submissions will be allowed until a tip extends the expiration. therefore no need to create a new query
//	but update the expiration time
func (k msgServer) Tip(goCtx context.Context, msg *types.MsgTip) (*types.MsgTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return nil, sdkerrors.ErrInvalidRequest
	}
	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)

	tip, err := k.keeper.transfer(ctx, tipper, msg.Amount)
	if err != nil {
		return nil, err
	}

	// get query id bytes hash from query data
	queryId := utils.QueryIDFromData(msg.QueryData)

	// get query info for the query id
	query, err := k.keeper.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		// initialize query tip first time
		query, err = k.keeper.InitializeQuery(ctx, msg.QueryData)
		if err != nil {
			return nil, err
		}

		query.Amount = math.ZeroInt()
		query.Expiration = uint64(ctx.BlockHeight()) + query.RegistrySpecBlockWindow
	}
	// prevAmt := query.Amount
	query.Amount = query.Amount.Add(tip.Amount)

	// expired submission window
	if query.Expiration <= uint64(ctx.BlockHeight()) {
		// query expired, create new expiration time
		query.Expiration = uint64(ctx.BlockHeight()) + query.RegistrySpecBlockWindow
		// if reporting window is expired that means the query is not in cycle
		query.CycleList = false
		// when report is expired and aggregated the query struct is removed
		// so when is this condition true?
		// when a cycle list query hasn't been reported and the time is expired (time=expiration+Offset)
		// and before it becomes in cycle a tip comes in then a new query is created to identify the tip
		// if prevAmt.IsZero() {
		// 	id, err := k.keeper.QuerySequencer.Next(ctx)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	query.Id = id
		// 	query.Amount = tip.Amount
		// 	query.HasRevealedReports = false

		// }
	}
	err = k.keeper.Query.Set(ctx, collections.Join(queryId, query.Id), query)
	if err != nil {
		return nil, err
	}

	// update totals
	if err := k.keeper.AddToTipperTotal(ctx, tipper, tip.Amount); err != nil {
		return nil, err
	}
	if err := k.keeper.AddtoTotalTips(ctx, tip.Amount); err != nil {
		return nil, err
	}
	// prevTip, err := k.keeper.Tips.Get(ctx, collections.Join(queryId, tipper.Bytes()))
	// if err != nil && !errors.Is(err, collections.ErrNotFound) {
	// 	return nil, fmt.Errorf("failed to get previous tip: %w", err)
	// }

	// if !prevTip.IsNil() {
	// 	tip = tip.AddAmount(prevTip)
	// }
	// err = k.keeper.Tips.Set(ctx, collections.Join(queryId, tipper.Bytes()), tip.Amount)
	// if err != nil {
	// 	return nil, err
	// }
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tip_added",
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("tipper", tipper.String()),
			sdk.NewAttribute("amount", tip.Amount.String()),
		),
	})
	return &types.MsgTipResponse{}, nil
}
