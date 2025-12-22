package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/tellor-io/layer/lib/metrics"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Tip handles tipping a query; accepts query data and amount to tip.
// 1. Checks if the bond denom is correct and if the amount is positive (in ValidateBasic).
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

	err := validateTip(msg)
	if err != nil {
		return nil, err
	}

	tipper := sdk.MustAccAddressFromBech32(msg.Tipper)
	params, err := k.keeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if msg.Amount.Amount.LT(params.MinTipAmount) {
		return nil, types.ErrNotEnoughTip
	} else if msg.Amount.Amount.GT(params.MaxTipAmount) {
		return nil, types.ErrTipExceedsMax
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

	// if an additional tip exceeds max tip, return an error
	if query.Amount.Add(msg.Amount.Amount).GT(types.DefaultMaxTipAmount) {
		return nil, types.ErrTipExceedsMax
	}

	// transfer the tip amount to the module account after burning 2% of the tip
	tip, err := k.keeper.transfer(ctx, tipper, msg.Amount)
	if err != nil {
		return nil, err
	}

	query.Amount = query.Amount.Add(tip.Amount)

	// expired submission window
	if query.Expiration < uint64(ctx.BlockHeight()) {
		// query expired, create new expiration time
		query.Expiration = uint64(ctx.BlockHeight()) + query.RegistrySpecBlockWindow

		// check if this is a cyclelist query being tipped out-of-turn
		isCyclelistQuery, _ := k.keeper.Cyclelist.Has(ctx, queryId)
		if isCyclelistQuery {
			// keep CycleList = true for liveness tracking
			query.CycleList = true
			// Demote query to non-standard (out-of-turn tip creates extra opportunity)
			// This moves existing shares from standard to non-standard tracking
			if err := k.keeper.DemoteQueryToNonStandard(ctx, queryId); err != nil {
				return nil, err
			}
			// increment query opportunities (creates extra opportunity)
			if err := k.keeper.IncrementQueryOpportunities(ctx, queryId); err != nil {
				return nil, err
			}
		} else {
			// non-cyclelist query, not tracked for liveness
			query.CycleList = false
		}
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

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tip_added",
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("tipper", tipper.String()),
			sdk.NewAttribute("amount", tip.Amount.String()),
			sdk.NewAttribute("querymeta_id", strconv.Itoa(int(query.Id))),
		),
	})

	defer func() {
		// track both the total tips for a query id and the amount of times that a a query id is tipped
		telemetry.IncrCounterWithLabels([]string{"oracle_tip_tracker"}, float32(tip.Amount.Uint64()), []metrics.Label{{Name: "chain_id", Value: ctx.ChainID()}, {Name: "query_id", Value: hex.EncodeToString(queryId)}})
		telemetry.IncrCounterWithLabels([]string{"oracle_tipped_query"}, 1, []metrics.Label{{Name: "chain_id", Value: ctx.ChainID()}, {Name: "query_id", Value: hex.EncodeToString(queryId)}})
	}()
	return &types.MsgTipResponse{}, nil
}

func validateTip(msg *types.MsgTip) error {
	_, err := sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	}
	// ensure that the msg.Amount.Denom matches the layer.BondDenom and the amount is a positive number
	if msg.Amount.Denom != layer.BondDenom || msg.Amount.Amount.IsZero() || msg.Amount.Amount.IsNegative() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid tip amount (%s)", msg.Amount.String())
	}
	// ensure that the queryData is not empty
	if len(msg.QueryData) == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data is empty")
	}
	return nil
}
