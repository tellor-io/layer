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

	tipper, err := validateTip(msg)
	if err != nil {
		return nil, err
	}

	tip, queryId, query, err := k.keeper.processTipCore(ctx, tipper, msg.QueryData, msg.Amount)
	if err != nil {
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

func (k Keeper) processTipCore(ctx sdk.Context, tipper sdk.AccAddress, queryData []byte, amount sdk.Coin) (sdk.Coin, []byte, types.QueryMeta, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdk.Coin{}, nil, types.QueryMeta{}, err
	}
	if amount.Amount.LT(params.MinTipAmount) {
		return sdk.Coin{}, nil, types.QueryMeta{}, types.ErrNotEnoughTip
	} else if amount.Amount.GT(params.MaxTipAmount) {
		return sdk.Coin{}, nil, types.QueryMeta{}, types.ErrTipExceedsMax
	}

	// get query id bytes hash from query data
	queryId := utils.QueryIDFromData(queryData)

	// get query info for the query id
	query, err := k.CurrentQuery(ctx, queryId)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return sdk.Coin{}, nil, types.QueryMeta{}, err
		}
		// initialize query tip first time
		query, err = k.InitializeQuery(ctx, queryData)
		if err != nil {
			return sdk.Coin{}, nil, types.QueryMeta{}, err
		}

		query.Amount = math.ZeroInt()
		query.Expiration = uint64(ctx.BlockHeight()) + query.RegistrySpecBlockWindow
	}

	// if an additional tip exceeds max tip, return an error
	if query.Amount.Add(amount.Amount).GT(types.DefaultMaxTipAmount) {
		return sdk.Coin{}, nil, types.QueryMeta{}, types.ErrTipExceedsMax
	}

	// transfer the tip amount to the module account after burning 2% of the tip
	tip, err := k.transfer(ctx, tipper, amount)
	if err != nil {
		return sdk.Coin{}, nil, types.QueryMeta{}, err
	}

	query.Amount = query.Amount.Add(tip.Amount)

	// expired submission window
	if query.Expiration < uint64(ctx.BlockHeight()) {
		// query expired, create new expiration time
		query.Expiration = uint64(ctx.BlockHeight()) + query.RegistrySpecBlockWindow

		// check if this is a cyclelist query being tipped out-of-turn
		isCyclelistQuery, _ := k.Cyclelist.Has(ctx, queryId)
		if isCyclelistQuery {
			// keep CycleList = true for liveness tracking
			query.CycleList = true
			// Demote query to non-standard (out-of-turn tip creates extra opportunity)
			// This moves existing shares from standard to non-standard tracking
			if err := k.DemoteQueryToNonStandard(ctx, queryId); err != nil {
				return sdk.Coin{}, nil, types.QueryMeta{}, err
			}
			// increment query opportunities (creates extra opportunity)
			if err := k.IncrementQueryOpportunities(ctx, queryId); err != nil {
				return sdk.Coin{}, nil, types.QueryMeta{}, err
			}
		} else {
			// non-cyclelist query, not tracked for liveness
			query.CycleList = false
		}

		id, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return sdk.Coin{}, nil, types.QueryMeta{}, err
		}
		// remove old query with old ID before creating new one with new ID
		oldId := query.Id
		err = k.Query.Remove(ctx, collections.Join(queryId, oldId))
		if err != nil {
			return sdk.Coin{}, nil, types.QueryMeta{}, err
		}
		query.Id = id
	}
	err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
	if err != nil {
		return sdk.Coin{}, nil, types.QueryMeta{}, err
	}

	// update totals
	if err := k.AddToTipperTotal(ctx, tipper, tip.Amount); err != nil {
		return sdk.Coin{}, nil, types.QueryMeta{}, err
	}
	if err := k.AddtoTotalTips(ctx, tip.Amount); err != nil {
		return sdk.Coin{}, nil, types.QueryMeta{}, err
	}

	return tip, queryId, query, nil
}

func validateTip(msg *types.MsgTip) (tipper sdk.AccAddress, err error) {
	tipper, err = sdk.AccAddressFromBech32(msg.Tipper)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid tipper address (%s)", err)
	}
	if err := validateTipFields(msg.Amount, msg.QueryData); err != nil {
		return nil, err
	}
	return tipper, nil
}

func validateTipFields(amount sdk.Coin, queryData []byte) error {
	// ensure that the amount denom matches the layer.BondDenom and the amount is a positive number
	if amount.Denom != layer.BondDenom || amount.Amount.IsZero() || amount.Amount.IsNegative() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "invalid tip amount (%s)", amount.String())
	}
	// ensure that the queryData is not empty
	if len(queryData) == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "query data is empty")
	}
	return nil
}
