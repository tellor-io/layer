package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreateReporter(goCtx context.Context, msg *types.MsgCreateReporter) (*types.MsgCreateReporterResponse, error) {
	// check if reporter has min bonded tokens
	addr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	bondedTokens, count, err := k.Keeper.CheckSelectorsDelegations(goCtx, addr)
	if err != nil {
		return nil, err
	}
	if params.MinTrb.GT(bondedTokens) {
		return nil, errors.New("address does not have min tokens required to be a reporter with a BONDED validator")
	}
	// the min requirement chosen by reporter has gte the min requirement
	if msg.MinTokensRequired.LT(params.MinTrb) {
		return nil, errors.New("reporters chosen min to join must be gte the min requirement")
	}
	// reporter can't be previously a selector or a reporter
	alreadyExists, err := k.Keeper.Selectors.Has(goCtx, addr)
	if err != nil {
		return nil, err
	}
	if alreadyExists {
		return nil, errors.New("address already exists")
	}

	if msg.CommissionRate.GT(math.NewUint(1e6)) {
		return nil, errors.New("commission rate must be below 1000000 as that is a 100 percent commission rate")
	}
	// set the reporter and set the self selector
	if err := k.Keeper.Reporters.Set(goCtx, addr.Bytes(), types.NewReporter(msg.CommissionRate, msg.MinTokensRequired)); err != nil {
		return nil, err
	}
	if err := k.Keeper.Selectors.Set(goCtx, addr.Bytes(), types.NewSelection(addr.Bytes(), uint64(count))); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"created_reporter",
			sdk.NewAttribute("reporter", msg.ReporterAddress),
			sdk.NewAttribute("commission", msg.CommissionRate.String()),
			sdk.NewAttribute("min_tokens_required", msg.MinTokensRequired.String()),
		),
	})
	return &types.MsgCreateReporterResponse{}, nil
}

func (k msgServer) SelectReporter(goCtx context.Context, msg *types.MsgSelectReporter) (*types.MsgSelectReporterResponse, error) {
	// check if selector exists
	addr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	alreadyExists, err := k.Keeper.Selectors.Has(goCtx, addr)
	if err != nil {
		return nil, err
	}
	if alreadyExists {
		return nil, errors.New("selector already exists")
	}
	// check if reporter exists
	reporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	reporter, err := k.Keeper.Reporters.Get(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	// check if reporter is capped at max selectors
	iter, err := k.Keeper.Selectors.Indexes.Reporter.MatchExact(goCtx, reporterAddr.Bytes())
	if err != nil {
		return nil, err
	}
	selectors, err := iter.FullKeys()
	if err != nil {
		return nil, err
	}
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	if len(selectors) >= int(params.MaxSelectors) {
		return nil, errors.New("reporter has reached max selectors")
	}
	// check if selector meets reporters min requirement
	bondedTokens, count, err := k.Keeper.CheckSelectorsDelegations(goCtx, addr)
	if err != nil {
		return nil, err
	}
	if reporter.MinTokensRequired.GT(bondedTokens) {
		return nil, fmt.Errorf("reporter's min requirement %s not met by selector", reporter.MinTokensRequired.String())
	}
	// set the selector
	if err := k.Keeper.Selectors.Set(goCtx, addr.Bytes(), types.NewSelection(reporterAddr.Bytes(), uint64(count))); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"reporter_selected",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("reporter", msg.ReporterAddress),
			sdk.NewAttribute("reporter_selector_count_increased", strconv.Itoa(len(selectors)+1)),
		),
	})
	return &types.MsgSelectReporterResponse{}, nil
}

func (k msgServer) SwitchReporter(goCtx context.Context, msg *types.MsgSwitchReporter) (*types.MsgSwitchReporterResponse, error) {
	addr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	// check if selector exists
	selector, err := k.Keeper.Selectors.Get(goCtx, addr)
	if err != nil {
		return nil, err
	}
	prevReporter := sdk.AccAddress(selector.Reporter)
	if bytes.Equal(selector.Reporter, addr.Bytes()) {
		return nil, errors.New("cannot switch reporter if selector is a reporter")
	}
	// check if reporter exists
	reporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	reporter, err := k.Keeper.Reporters.Get(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	// check if reporter is capped at max selectors
	// todo: add field to reporter and to selectors to keep track of how many selectors have and for selectors an id
	iter, err := k.Keeper.Selectors.Indexes.Reporter.MatchExact(goCtx, reporterAddr.Bytes())
	if err != nil {
		return nil, err
	}
	selectors, err := iter.FullKeys()
	if err != nil {
		return nil, err
	}
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	if len(selectors) >= int(params.MaxSelectors) {
		return nil, errors.New("reporter has reached max selectors")
	}
	// check if selector meets reporters min requirement
	hasMin, err := k.Keeper.HasMin(goCtx, addr, reporter.MinTokensRequired)
	if err != nil {
		return nil, err
	}
	if !hasMin {
		return nil, fmt.Errorf("reporter's min requirement %s not met by selector", reporter.MinTokensRequired.String())
	}

	// check if selector was part of a report before switching
	var prevReportedPower math.Int
	rng := collections.NewPrefixedPairRange[[]byte, uint64](selector.Reporter).EndInclusive(uint64(sdk.UnwrapSDKContext(goCtx).BlockHeight())).Descending()
	err = k.Keeper.Report.Walk(goCtx, rng, func(_ collections.Pair[[]byte, uint64], value types.DelegationsAmounts) (stop bool, err error) {
		prevReportedPower = value.Total
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	if !prevReportedPower.IsNil() {
		unbondingTime, err := k.stakingKeeper.UnbondingTime(goCtx)
		if err != nil {
			return nil, err
		}

		selector.LockedUntilTime = sdk.UnwrapSDKContext(goCtx).BlockTime().Add(unbondingTime)
	}
	selector.Reporter = reporterAddr.Bytes()
	if err := k.Keeper.Selectors.Set(goCtx, addr.Bytes(), selector); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"switched_reporter",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("previous_reporter", prevReporter.String()),
			sdk.NewAttribute("new_reporter", msg.ReporterAddress),
			sdk.NewAttribute("selector_locked_until", selector.LockedUntilTime.String()),
		),
	})
	return &types.MsgSwitchReporterResponse{}, nil
}

func (k msgServer) RemoveSelector(goCtx context.Context, msg *types.MsgRemoveSelector) (*types.MsgRemoveSelectorResponse, error) {
	selectorAddr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	selector, err := k.Keeper.Selectors.Get(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	reporter, err := k.Keeper.Reporters.Get(goCtx, selector.Reporter)
	if err != nil {
		return nil, err
	}

	hasMin, err := k.Keeper.HasMin(goCtx, selectorAddr, reporter.MinTokensRequired)
	if err != nil {
		return nil, err
	}
	if hasMin {
		return nil, errors.New("selector can't be removed if reporter's min requirement is met")
	}

	if !hasMin {
		params, err := k.Keeper.Params.Get(goCtx)
		if err != nil {
			return nil, err
		}
		// check if reporter is capped if not need to remove selector.
		iter, err := k.Keeper.Selectors.Indexes.Reporter.MatchExact(goCtx, selector.Reporter)
		if err != nil {
			return nil, err
		}
		selectors, err := iter.FullKeys()
		if err != nil {
			return nil, err
		}
		if len(selectors) <= int(params.MaxSelectors) {
			return nil, errors.New("selector can only be removed if reporter has reached max selectors and doesn't meet min requirement")
		}
	}
	// remove selector
	if err := k.Keeper.Selectors.Remove(goCtx, selectorAddr); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"selector_removed",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("removed_from_reporter", sdk.AccAddress(selector.Reporter).String()),
		),
	})
	return &types.MsgRemoveSelectorResponse{}, nil
}

func (k msgServer) UnjailReporter(goCtx context.Context, msg *types.MsgUnjailReporter) (*types.MsgUnjailReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)

	reporter, err := k.Reporters.Get(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	if err := k.Keeper.UnjailReporter(ctx, reporterAddr, reporter); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"unjailed_reporter",
			sdk.NewAttribute("reporter", reporterAddr.String()),
		),
	})
	return &types.MsgUnjailReporterResponse{}, nil
}

func (k msgServer) WithdrawTip(goCtx context.Context, msg *types.MsgWithdrawTip) (*types.MsgWithdrawTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	delAddr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	shares, err := k.Keeper.SelectorTips.Get(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if !val.IsBonded() {
		return nil, errors.New("chosen validator must be bonded")
	}
	amtToDelegate := shares.Value.QuoUint64(1e6)
	if amtToDelegate.IsZero() {
		return nil, errors.New("no tips to withdraw")
	}
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, delAddr, math.NewInt(int64(amtToDelegate.Uint64())), val.Status, val, false)
	if err != nil {
		return nil, err
	}

	remainder := shares.Value.Sub(amtToDelegate.MulUint64(1e6))
	if remainder.IsZero() {
		err = k.Keeper.SelectorTips.Remove(ctx, delAddr)
		if err != nil {
			return nil, err
		}
	} else {
		err = k.Keeper.SelectorTips.Set(ctx, delAddr, types.BigUint{Value: remainder})
		if err != nil {
			return nil, err
		}
	}

	// send coins
	err = k.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.TipsEscrowPool, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, math.NewInt(int64(amtToDelegate.Uint64())))))
	if err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"tip_withdrawn",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("validator", msg.ValidatorAddress),
			sdk.NewAttribute("amount", amtToDelegate.String()),
		),
	})
	return &types.MsgWithdrawTipResponse{}, nil
}
