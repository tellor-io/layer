package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tellor-io/layer/lib/metrics"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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
	err := validateCreateReporter(msg)
	if err != nil {
		return nil, err
	}
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
	if params.MinLoya.GT(bondedTokens) {
		return nil, errors.New("address does not have min tokens required to be a reporter staked with a BONDED validator")
	}
	// the min requirement chosen by reporter has gte the min requirement
	if msg.MinTokensRequired.LT(params.MinLoya) {
		return nil, errors.New("reporters chosen min tokens for selectors to join must be gte the min requirement")
	}
	// reporter commission rate must be between 0 and 1
	if msg.CommissionRate.GT(math.LegacyNewDec(1)) || msg.CommissionRate.LT(params.MinCommissionRate) {
		return nil, errors.New("commission rate must be between 0 and 1 (e.g, 0.50 = 50%)")
	}
	// reporter can't be previously a reporter
	alreadyExists, err := k.Keeper.Selectors.Has(goCtx, addr)
	if err != nil {
		return nil, err
	}
	if alreadyExists {
		// check if they are a reporter already
		selection, err := k.Keeper.Selectors.Get(goCtx, addr)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(selection.Reporter, addr.Bytes()) {
			return nil, errors.New("address is already a reporter")
		}
		// check if selector was part of a report before switching
		prevReporter := sdk.AccAddress(selection.Reporter)
		prevReportedPower, err := k.Keeper.GetReporterTokensAtBlock(goCtx, prevReporter, uint64(sdk.UnwrapSDKContext(goCtx).BlockHeight()))
		if err != nil {
			return nil, err
		}
		if !prevReportedPower.IsZero() {
			unbondingTime, err := k.stakingKeeper.UnbondingTime(goCtx)
			if err != nil {
				return nil, err
			}
			selection.LockedUntilTime = sdk.UnwrapSDKContext(goCtx).BlockTime().Add(unbondingTime)
		}
		selection.Reporter = addr.Bytes()
		if err := k.Keeper.Selectors.Set(goCtx, addr.Bytes(), selection); err != nil {
			return nil, err
		}
		if err := k.Keeper.Reporters.Set(goCtx, addr.Bytes(), types.NewReporter(msg.CommissionRate, msg.MinTokensRequired, msg.Moniker)); err != nil {
			return nil, err
		}
		sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"created_reporter_from_selector",
				sdk.NewAttribute("reporter", msg.ReporterAddress),
				sdk.NewAttribute("commission", msg.CommissionRate.String()),
				sdk.NewAttribute("min_tokens_required", msg.MinTokensRequired.String()),
				sdk.NewAttribute("moniker", msg.Moniker),
			),
		})
		telemetry.IncrCounterWithLabels([]string{"create_reporter_count"}, 1, []metrics.Label{{Name: "chain_id", Value: sdk.UnwrapSDKContext(goCtx).ChainID()}})
		return &types.MsgCreateReporterResponse{}, nil
	}

	// set the reporter and set the self selector
	if err := k.Keeper.Reporters.Set(goCtx, addr.Bytes(), types.NewReporter(msg.CommissionRate, msg.MinTokensRequired, msg.Moniker)); err != nil {
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
			sdk.NewAttribute("moniker", msg.Moniker),
		),
	})
	telemetry.IncrCounterWithLabels([]string{"create_reporter_count"}, 1, []metrics.Label{{Name: "chain_id", Value: sdk.UnwrapSDKContext(goCtx).ChainID()}})
	return &types.MsgCreateReporterResponse{}, nil
}

func validateCreateReporter(msg *types.MsgCreateReporter) error {
	_, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}

	// check that mintokensrequired is positive
	if msg.MinTokensRequired.LTE(math.ZeroInt()) {
		return errors.New("MinTokensRequired must be positive (%s)")
	}

	// check that moniker is not empty
	if msg.Moniker == "" {
		return errors.New("moniker cannot be empty")
	}
	return nil
}

// Msg: SelectReporter, allows a selector to join a reporter if they meet the min requirement set by the reporter
// and the reporter has not reached the max selectors allowed
// selector can only join one reporter at a time and to switch reporters see SwitchReporter
func (k msgServer) SelectReporter(goCtx context.Context, msg *types.MsgSelectReporter) (*types.MsgSelectReporterResponse, error) {
	err := validateSelectReporter(msg)
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("reporter's min requirement %s not met by selector. Must stake %s more to select to this reporter", reporter.MinTokensRequired.String(), reporter.MinTokensRequired.Sub(bondedTokens).String())
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
	telemetry.IncrCounterWithLabels([]string{"num_of_selectors", "join"}, 1, []metrics.Label{{Name: "chain_id", Value: sdk.UnwrapSDKContext(goCtx).ChainID()}})
	return &types.MsgSelectReporterResponse{}, nil
}

func validateSelectReporter(msg *types.MsgSelectReporter) error {
	_, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	return nil
}

// Msg: SwitchReporter, allows a selector to switch reporters if they meet the new reporters min requirement
// and the new reporter has not reached the max selectors allowed
// switching reporters will not automatically include the selector's tokens to be part of reporting until the unbonding time has passed
// in order to prevent the selector from being part of a report twice unless they were part of a reporter that hasn't reported yet
func (k msgServer) SwitchReporter(goCtx context.Context, msg *types.MsgSwitchReporter) (*types.MsgSwitchReporterResponse, error) {
	err := validateSwitchReporter(msg)
	if err != nil {
		return nil, err
	}
	addr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	// check if selector exists
	selector, err := k.Keeper.Selectors.Get(goCtx, addr)
	if err != nil {
		return nil, err
	}
	prevReporter := sdk.AccAddress(selector.Reporter)
	// check if reporter exists
	reporterAddr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	reporter, err := k.Keeper.Reporters.Get(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	// check if reporter is trying to become a selector, can only switch if havent reported in the last 21 days
	if bytes.Equal(selector.Reporter, addr.Bytes()) {
		// get the most recent report for the reporter
		microReport, err := k.oracleKeeper.GetMostRecentReport(goCtx, reporterAddr)
		if err != nil && err.Error() != "no reports found" {
			return nil, err
		}
		lastReportTimestamp := microReport.Timestamp
		if lastReportTimestamp.Add(21 * 24 * time.Hour).After(sdk.UnwrapSDKContext(goCtx).BlockTime()) {
			return nil, errors.New("reporter has reported in the last 21 days, please wait before switching reporters")
		}

		if err := k.Keeper.Reporters.Remove(goCtx, addr.Bytes()); err != nil {
			return nil, err
		}
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
		return nil, fmt.Errorf("reporter's min requirement of %s not met by selector. Must stake enough to reach the minimum", reporter.MinTokensRequired.String())
	}

	// check if selector was part of a report before switching
	prevReportedPower, err := k.Keeper.GetReporterTokensAtBlock(goCtx, sdk.MustAccAddressFromBech32(prevReporter.String()), uint64(sdk.UnwrapSDKContext(goCtx).BlockHeight()))
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

func validateSwitchReporter(msg *types.MsgSwitchReporter) error {
	_, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	return nil
}

// Msg: RemoveSelector, allows anyone to remove a selector if the selector falls below a given reporter's min requirement in order to free up space for new selectors
// if they are capped at max selectors
func (k msgServer) RemoveSelector(goCtx context.Context, msg *types.MsgRemoveSelector) (*types.MsgRemoveSelectorResponse, error) {
	err := validateRemoveSelector(msg)
	if err != nil {
		return nil, err
	}
	selectorAddr := sdk.MustAccAddressFromBech32(msg.SelectorAddress)
	selector, err := k.Keeper.Selectors.Get(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	reporter, err := k.Keeper.Reporters.Get(goCtx, selector.Reporter)
	if err != nil {
		return nil, err
	}

	// ensure that a selector cannot be removed if it is the reporter’s own address
	if bytes.Equal(selector.Reporter, selectorAddr.Bytes()) {
		return nil, errors.New("selector cannot be removed if it is the reporter's own address")
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
		if len(selectors) < int(params.MaxSelectors) {
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

func validateRemoveSelector(msg *types.MsgRemoveSelector) error {
	_, err := sdk.AccAddressFromBech32(msg.AnyAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	return nil
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
	err := validateWithdrawTip(msg)
	if err != nil {
		return nil, err
	}
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
	amtToDelegate := shares.TruncateInt()
	if amtToDelegate.IsZero() {
		return nil, errors.New("no tips to withdraw")
	}
	_, err = k.Keeper.stakingKeeper.Delegate(ctx, delAddr, amtToDelegate, val.Status, val, false)
	if err != nil {
		return nil, err
	}

	// isolate decimals from shares
	remainder := shares.Sub(shares.TruncateDec())
	if remainder.IsZero() {
		err = k.Keeper.SelectorTips.Remove(ctx, delAddr)
		if err != nil {
			return nil, err
		}
	} else {
		err = k.Keeper.SelectorTips.Set(ctx, delAddr, remainder)
		if err != nil {
			return nil, err
		}
	}

	// send coins
	escrowPoolAddr := k.Keeper.accountKeeper.GetModuleAddress(types.TipsEscrowPool)
	err = k.Keeper.bankKeeper.DelegateCoinsFromAccountToModule(ctx, escrowPoolAddr, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layertypes.BondDenom, math.NewInt(int64(amtToDelegate.Uint64())))))
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
	// allow for people to track the amount they have withdrawn based on their address
	telemetry.IncrCounterWithLabels([]string{"withdrawn_amount_tracker"}, float32(amtToDelegate.Int64()), []metrics.Label{{Name: "chain_id", Value: ctx.ChainID()}, {Name: "reporter", Value: hex.EncodeToString(delAddr.Bytes())}})
	return &types.MsgWithdrawTipResponse{}, nil
}

func validateWithdrawTip(msg *types.MsgWithdrawTip) error {
	_, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}

func (k msgServer) EditReporter(goCtx context.Context, msg *types.MsgEditReporter) (*types.MsgEditReporterResponse, error) {
	err := validateEditReporter(msg)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	addr := sdk.MustAccAddressFromBech32(msg.ReporterAddress)
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}

	// reporter commission rate must be between 0 and 1
	if msg.CommissionRate.GT(math.LegacyNewDec(1)) || msg.CommissionRate.LT(params.MinCommissionRate) {
		return nil, errors.New("commission rate must be between 0 and 1 (e.g, 0.50 = 50%)")
	}

	reporter, err := k.Keeper.Reporter(goCtx, addr)
	if err != nil {
		return nil, err
	}

	if sdkCtx.BlockTime().Sub(reporter.LastUpdated).Seconds() < 12*60*60 {
		return nil, errors.New("can only update reporters every 12 hours")
	}

	rateDiff := reporter.CommissionRate.Sub(msg.CommissionRate).Abs()
	if rateDiff.GT(math.LegacyMustNewDecFromStr("0.01")) {
		return nil, errors.New("commission rate cannot change by more than 1%")
	}

	minTokensRequiredDiff := msg.MinTokensRequired.Sub(reporter.MinTokensRequired).Abs()
	if math.LegacyNewDecFromInt(minTokensRequiredDiff).Quo(math.LegacyNewDecFromInt(reporter.MinTokensRequired)).GT(math.LegacyMustNewDecFromStr("0.10")) {
		return nil, errors.New("MinTokensRequired cannot change more than 10%")
	}

	reporter.CommissionRate = msg.CommissionRate
	reporter.MinTokensRequired = msg.MinTokensRequired
	reporter.Moniker = msg.Moniker
	reporter.LastUpdated = sdkCtx.BlockTime()

	err = k.Keeper.Reporters.Set(goCtx, addr.Bytes(), reporter)
	if err != nil {
		return nil, err
	}

	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"edited_reporter",
			sdk.NewAttribute("reporter", msg.ReporterAddress),
			sdk.NewAttribute("commission", msg.CommissionRate.String()),
			sdk.NewAttribute("min_tokens_required", msg.MinTokensRequired.String()),
			sdk.NewAttribute("moniker", msg.Moniker),
		),
	})

	return &types.MsgEditReporterResponse{}, nil
}

func validateEditReporter(msg *types.MsgEditReporter) error {
	_, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	// check that mintokensrequired is positive
	if msg.MinTokensRequired.LTE(math.ZeroInt()) {
		return errors.New("MinTokensRequired must be positive (%s)")
	}

	// check that moniker is not empty
	if msg.Moniker == "" {
		return errors.New("moniker cannot be empty")
	}

	return nil
}
