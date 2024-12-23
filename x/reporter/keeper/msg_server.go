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

// Msg: CreateReporter, adds a new reporter if it was never registered before and meets the min bonded tokens requirement
// allows the reporter to set their commission rate and min tokens required for selectors to join
func (k msgServer) CreateReporter(goCtx context.Context, msg *types.MsgCreateReporter) (*types.MsgCreateReporterResponse, error) {
	// check if reporter has min bonded tokens
	addr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	// calculate the bonded tokens for the given reporter address that is BONDED in the staking module
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

	if msg.CommissionRate.GT(math.LegacyOneDec()) {
		return nil, errors.New("commission rate must be LTE 1 as that is a 100 percent commission rate")
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

// Msg: SelectReporter, allows a selector to join a reporter if they meet the min requirement set by the reporter
// and the reporter has not reached the max selectors allowed
// selector can only join one reporter at a time and to switch reporters see SwitchReporter
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
	// count the selectors BONDED tokens in the staking module
	bondedTokens, count, err := k.Keeper.CheckSelectorsDelegations(goCtx, addr)
	if err != nil {
		return nil, err
	}
	// check if selector meets reporters min requirement
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

// Msg: SwitchReporter, allows a selector to switch reporters if they meet the new reporters min requirement
// and the new reporter has not reached the max selectors allowed
// switching reporters will not automatically include the selector's tokens to be part of reporting until the unbonding time has passed
// in order to prevent the selector from being part of a report twice unless they were part of a reporter that hasn't reported yet
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
	prevReportedPower, err := k.Keeper.GetReporterTokensAtBlock(goCtx, selector.Reporter, uint64(sdk.UnwrapSDKContext(goCtx).BlockHeight()))
	if err != nil {
		return nil, err
	}

	if !prevReportedPower.IsZero() {
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

// Msg: RemoveSelector, allows anyone to remove a selector if the selector falls below a given reporter's min requirement in order to free up space for new selectors
// if they are capped at max selectors
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

// Msg: UnjailReporter, allows a reporter that is jailed to be unjailed if the jail period has passed (jail period is set during a dispute)
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

// Msg: WithdrawTip, allows selectors to directly withdraw reporting rewards and stake them with a BONDED validator
func (k msgServer) WithdrawTip(goCtx context.Context, msg *types.MsgWithdrawTip) (*types.MsgWithdrawTipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	selectorAddr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	repAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)

	shares, err := k.Keeper.RewardByReporter(ctx, selectorAddr, repAddr, msg.Id, msg.QueryId)
	if err != nil {
		return nil, err
	}
	if shares.IsZero() {
		return nil, errors.New("no tips to withdraw")
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

	amtToDelegate := shares.TruncateInt()
	if amtToDelegate.IsZero() {
		return nil, errors.New("no tips to withdraw")
	}
	// remainder ie .1234 = 1.1234 - 1
	remainder := shares.Sub(shares.TruncateDec())
	prevRemainder, err := k.Keeper.SelectorTips.Get(ctx, selectorAddr)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	if prevRemainder.IsNil() {
		prevRemainder = math.LegacyZeroDec()
	}

	if remainder.IsPositive() {
		newRemainder := prevRemainder.Add(remainder)
		whole := newRemainder.TruncateInt()
		if whole.IsPositive() {
			amtToDelegate = amtToDelegate.Add(whole)
			newRemainder = newRemainder.Sub(newRemainder.TruncateDec())
		}
		if newRemainder.IsPositive() {
			err = k.Keeper.SelectorTips.Set(ctx, selectorAddr, newRemainder)
		} else {
			err = k.Keeper.SelectorTips.Remove(ctx, selectorAddr)
		}
		if err != nil {
			return nil, err
		}
	}
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, selectorAddr, amtToDelegate, val.Status, val, false)
	if err != nil {
		return nil, err
	}
	// send coins
	err = k.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx,
		types.TipsEscrowPool, stakingtypes.BondedPoolName,
		sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, amtToDelegate)),
	)
	if err != nil {
		return nil, err
	}
	err = k.ClaimStatus.Set(ctx, collections.Join(selectorAddr.Bytes(), msg.Id), true)
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
