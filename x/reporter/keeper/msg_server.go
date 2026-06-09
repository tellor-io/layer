package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/tellor-io/layer/lib/metrics"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const TwentyOneDaysInMs = 21 * 24 * 60 * 60 * 1000

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
	addr, err := validateCreateReporter(msg)
	if err != nil {
		return nil, err
	}
	// check if reporter has min bonded tokens
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
		hasPending, err := k.Keeper.hasOutgoingPendingSwitch(goCtx, selection.Reporter, addr.Bytes())
		if err != nil {
			return nil, err
		}
		prevReporter := sdk.AccAddress(selection.Reporter)
		sdkCtx := sdk.UnwrapSDKContext(goCtx)

		if !hasPending {
			if selection.SwitchOutLockedUntilBlock >= uint64(sdkCtx.BlockHeight()) && selection.SwitchOutLockedUntilBlock != 0 {
				return nil, errors.New("selector is locked until the current reporter switch completes")
			}
		}

		if err := k.Keeper.scheduleReporterSwitch(goCtx, addr, &selection, prevReporter, addr); err != nil {
			return nil, err
		}

		if err := k.Keeper.Reporters.Set(goCtx, addr.Bytes(), types.NewReporter(msg.CommissionRate, msg.MinTokensRequired, msg.Moniker)); err != nil {
			return nil, err
		}

		selAfter, err := k.Keeper.Selectors.Get(goCtx, addr.Bytes())
		if err != nil {
			return nil, err
		}
		maxExp := selAfter.SwitchOutLockedUntilBlock
		sdkCtx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(
				"created_reporter_from_selector",
				sdk.NewAttribute("reporter", msg.ReporterAddress),
				sdk.NewAttribute("commission", msg.CommissionRate.String()),
				sdk.NewAttribute("min_tokens_required", msg.MinTokensRequired.String()),
				sdk.NewAttribute("moniker", msg.Moniker),
				sdk.NewAttribute("pending_switch_lock_until_block", strconv.FormatUint(maxExp, 10)),
			),
		})
		if err := k.Keeper.FlagStakeRecalc(goCtx, prevReporter); err != nil {
			return nil, err
		}
		if err := k.Keeper.FlagStakeRecalc(goCtx, addr); err != nil {
			return nil, err
		}
		telemetry.IncrCounterWithLabels([]string{"create_reporter_count"}, 1, []metrics.Label{{Name: "chain_id", Value: sdkCtx.ChainID()}})
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

func validateCreateReporter(msg *types.MsgCreateReporter) (reporter sdk.AccAddress, err error) {
	reporter, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}

	// check that mintokensrequired is positive
	if msg.MinTokensRequired.LTE(math.ZeroInt()) {
		return nil, errors.New("MinTokensRequired must be positive (%s)")
	}

	// check that moniker is not empty
	if msg.Moniker == "" {
		return nil, errors.New("moniker cannot be empty")
	}
	return reporter, nil
}

// Msg: SelectReporter, allows a selector to join a reporter if they meet the min requirement set by the reporter
// and the reporter has not reached the max selectors allowed
// selector can only join one reporter at a time and to switch reporters see SwitchReporter
func (k msgServer) SelectReporter(goCtx context.Context, msg *types.MsgSelectReporter) (*types.MsgSelectReporterResponse, error) {
	selectorAddr, reporterAddr, err := validateSelectReporter(msg)
	if err != nil {
		return nil, err
	}
	// check if selector exists
	alreadyExists, err := k.Keeper.Selectors.Has(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	if alreadyExists {
		return nil, errors.New("selector already exists")
	}
	// check if reporter exists
	reporter, err := k.Keeper.Reporters.Get(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	// check if reporter is capped at max selectors (include incoming pending switches)
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	selectorCount, err := k.Keeper.GetNumOfSelectorsIncludingPendingIncoming(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	if selectorCount >= int(params.MaxSelectors) {
		return nil, errors.New("reporter has reached max selectors")
	}
	// count the selectors BONDED tokens in the staking module
	bondedTokens, count, err := k.Keeper.CheckSelectorsDelegations(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	// check if selector meets reporters min requirement
	if reporter.MinTokensRequired.GT(bondedTokens) {
		return nil, fmt.Errorf("reporter's min requirement %s not met by selector. Must stake %s more to select to this reporter", reporter.MinTokensRequired.String(), reporter.MinTokensRequired.Sub(bondedTokens).String())
	}
	// set the selector
	if err := k.Keeper.Selectors.Set(goCtx, selectorAddr.Bytes(), types.NewSelection(reporterAddr.Bytes(), uint64(count))); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(goCtx).EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"reporter_selected",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("reporter", msg.ReporterAddress),
			sdk.NewAttribute("reporter_selector_count_increased", strconv.Itoa(selectorCount+1)),
		),
	})
	telemetry.IncrCounterWithLabels([]string{"num_of_selectors", "join"}, 1, []metrics.Label{{Name: "chain_id", Value: sdk.UnwrapSDKContext(goCtx).ChainID()}})
	if err := k.Keeper.FlagStakeRecalc(goCtx, reporterAddr); err != nil {
		return nil, err
	}
	return &types.MsgSelectReporterResponse{}, nil
}

func validateSelectReporter(msg *types.MsgSelectReporter) (selector, reporter sdk.AccAddress, err error) {
	selector, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	reporter, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	return selector, reporter, nil
}

// Msg: SwitchReporter schedules a move to another reporter: a pending row is stored
// under the outgoing reporter, Selection.reporter stays on the outgoing address until
// unlock height, and ReporterStake (e.g. via MsgSubmitValue) applies the handoff.
// The selector's stake stops counting toward the outgoing reporter immediately; it
// does not count toward the incoming reporter until finalization. Caps, min stake, and
// oracle snapshot unlock (switch_out_locked_until_block) apply when not already
// in-flight for this selector.
func (k msgServer) SwitchReporter(goCtx context.Context, msg *types.MsgSwitchReporter) (*types.MsgSwitchReporterResponse, error) {
	selectorAddr, reporterAddr, err := validateSwitchReporter(msg)
	if err != nil {
		return nil, err
	}
	// check if selector exists
	selector, err := k.Keeper.Selectors.Get(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(selector.Reporter, reporterAddr.Bytes()) {
		return nil, errors.New("selector is already assigned to this reporter")
	}
	prevReporter := sdk.AccAddress(selector.Reporter)
	// check if reporter exists
	reporter, err := k.Keeper.Reporters.Get(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	pending, toB, err := k.Keeper.pendingSwitchToReporter(goCtx, prevReporter, selectorAddr)
	if err != nil {
		return nil, err
	}
	if pending && bytes.Equal(toB, reporterAddr.Bytes()) {
		return &types.MsgSwitchReporterResponse{}, nil
	}
	// check if reporter is trying to become a selector of another reporter: if they
	// still have other selectors, require 21 days since their last oracle report.
	if bytes.Equal(selector.Reporter, selectorAddr.Bytes()) {
		others, err := k.Keeper.CountSelectorsDelegatingToReporterExcludingSelf(goCtx, selectorAddr)
		if err != nil {
			return nil, err
		}
		if others > 0 {
			lastReportTimestamp, err := k.Keeper.oracleKeeper.GetLastReportedAtTimestamp(goCtx, selectorAddr.Bytes())
			if err != nil {
				return nil, err
			}
			currentBlocktime := uint64(sdk.UnwrapSDKContext(goCtx).BlockTime().UnixMilli())
			if currentBlocktime-lastReportTimestamp < TwentyOneDaysInMs {
				return nil, errors.New("reporter has other selectors; must wait 21 days since last report before delegating reporting to another reporter")
			}
		}

		maxCommit, err := k.Keeper.oracleKeeper.GetMaxOpenCommitmentForReporter(goCtx, selectorAddr.Bytes())
		if err != nil {
			return nil, err
		}
		currentBlock := uint64(sdk.UnwrapSDKContext(goCtx).BlockHeight())
		if maxCommit >= currentBlock {
			return nil, errors.New("cannot self-demote while reporter has open query commitments; wait until block height exceeds max open commitment height")
		}

		selfRep, selfErr := k.Keeper.Reporters.Get(goCtx, selectorAddr.Bytes())
		if selfErr == nil && selfRep.Jailed {
			if err := k.Keeper.copyReporterJailToSelection(goCtx, selectorAddr, selfRep); err != nil {
				return nil, err
			}
		} else if selfErr != nil && !errors.Is(selfErr, collections.ErrNotFound) {
			return nil, selfErr
		}
		if err := k.Keeper.Reporters.Remove(goCtx, selectorAddr.Bytes()); err != nil {
			return nil, err
		}
	}
	// check if reporter is capped at max selectors (include incoming pending switches)
	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}
	selectorCount, err := k.Keeper.GetNumOfSelectorsIncludingPendingIncoming(goCtx, reporterAddr)
	if err != nil {
		return nil, err
	}
	if selectorCount >= int(params.MaxSelectors) {
		return nil, errors.New("reporter has reached max selectors")
	}
	// check if selector meets reporters min requirement
	hasMin, err := k.Keeper.HasMin(goCtx, selectorAddr, reporter.MinTokensRequired)
	if err != nil {
		return nil, err
	}
	if !hasMin {
		return nil, fmt.Errorf("reporter's min requirement of %s not met by selector. Must stake enough to reach the minimum", reporter.MinTokensRequired.String())
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	currentBlock := uint64(sdkCtx.BlockHeight())

	hasPending, err := k.Keeper.hasOutgoingPendingSwitch(goCtx, prevReporter.Bytes(), selectorAddr.Bytes())
	if err != nil {
		return nil, err
	}
	if !hasPending {
		if selector.SwitchOutLockedUntilBlock >= currentBlock && selector.SwitchOutLockedUntilBlock != 0 {
			return nil, errors.New("selector is locked until the current reporter switch completes")
		}
	}

	// Original reporter must recompute stake immediately so the selector's power
	// is excluded from future reports while the switch is pending.
	if err := k.Keeper.FlagStakeRecalc(goCtx, prevReporter); err != nil {
		return nil, err
	}

	if err := k.Keeper.lazyClearSelectorLocksIfExpired(goCtx, selectorAddr, &selector); err != nil {
		return nil, err
	}
	if err := k.Keeper.scheduleReporterSwitch(goCtx, selectorAddr, &selector, prevReporter, reporterAddr); err != nil {
		return nil, err
	}

	selAfter, err := k.Keeper.Selectors.Get(goCtx, selectorAddr.Bytes())
	if err != nil {
		return nil, err
	}
	maxExp := selAfter.SwitchOutLockedUntilBlock
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"switched_reporter",
			sdk.NewAttribute("selector", msg.SelectorAddress),
			sdk.NewAttribute("previous_reporter", prevReporter.String()),
			sdk.NewAttribute("new_reporter", msg.ReporterAddress),
			sdk.NewAttribute("pending_switch_lock_until_block", strconv.FormatUint(maxExp, 10)),
		),
	})
	if err := k.Keeper.FlagStakeRecalc(goCtx, reporterAddr); err != nil {
		return nil, err
	}
	return &types.MsgSwitchReporterResponse{}, nil
}

func validateSwitchReporter(msg *types.MsgSwitchReporter) (selector, reporter sdk.AccAddress, err error) {
	selector, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	reporter, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	if bytes.Equal(selector.Bytes(), reporter.Bytes()) {
		return nil, nil, errors.New("selector and reporter cannot be the same address")
	}
	return selector, reporter, nil
}

// Msg: RemoveSelector, allows anyone to remove a selector if the selector falls below a given reporter's min requirement in order to free up space for new selectors
// if they are capped at max selectors
func (k msgServer) RemoveSelector(goCtx context.Context, msg *types.MsgRemoveSelector) (*types.MsgRemoveSelectorResponse, error) {
	selectorAddr, err := validateRemoveSelector(msg)
	if err != nil {
		return nil, err
	}
	selector, err := k.Keeper.Selectors.Get(goCtx, selectorAddr)
	if err != nil {
		return nil, err
	}
	reporter, err := k.Keeper.Reporters.Get(goCtx, selector.Reporter)
	if err != nil {
		return nil, err
	}

	// ensure that a selector cannot be removed if it is the reporter's own address
	if bytes.Equal(selector.Reporter, selectorAddr.Bytes()) {
		return nil, errors.New("selector cannot be removed if it is the reporter's own address")
	}

	if err := k.Keeper.maybeFinalizePendingSwitchForRemoveSelector(goCtx, selectorAddr, &selector, &reporter); err != nil {
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
		if len(selectors) < int(params.MaxSelectors) {
			return nil, errors.New("selector can only be removed if reporter has reached max selectors and doesn't meet min requirement")
		}
	}

	// remove selector
	if err := k.Keeper.Selectors.Remove(goCtx, selectorAddr); err != nil {
		return nil, err
	}
	if err := k.Keeper.FlagStakeRecalc(goCtx, sdk.AccAddress(selector.Reporter)); err != nil {
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

func validateRemoveSelector(msg *types.MsgRemoveSelector) (selector sdk.AccAddress, err error) {
	_, err = sdk.AccAddressFromBech32(msg.AnyAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	selector, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	return selector, nil
}

// Msg: UnjailReporter allows a jailed reporter or selector to be unjailed after their
// sentence. The reporter may unjail themselves once eligible; any account may unjail them
// seven days after that.
func (k msgServer) UnjailReporter(goCtx context.Context, msg *types.MsgUnjailReporter) (*types.MsgUnjailReporterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	callerAddr, reporterAddr, err := validateUnjailReporter(msg)
	if err != nil {
		return nil, err
	}

	if err := k.Keeper.UnjailReporter(ctx, callerAddr, reporterAddr); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"unjailed_reporter",
			sdk.NewAttribute("reporter", reporterAddr.String()),
			sdk.NewAttribute("caller", callerAddr.String()),
		),
	})
	return &types.MsgUnjailReporterResponse{}, nil
}

func validateUnjailReporter(msg *types.MsgUnjailReporter) (caller, reporter sdk.AccAddress, err error) {
	caller, err = sdk.AccAddressFromBech32(msg.SignerAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	reporter, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}
	return caller, reporter, nil
}

// Msg: WithdrawTip, allows selectors to directly withdraw reporting rewards and stake them with a BONDED validator
func (k msgServer) WithdrawTip(goCtx context.Context, msg *types.MsgWithdrawTip) (*types.MsgWithdrawTipResponse, error) {
	selectorAddr, err := validateWithdrawTip(msg)
	if err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the selector's reporter and settle any pending rewards
	selection, err := k.Keeper.Selectors.Get(ctx, selectorAddr)
	if err == nil {
		// Selector exists - settle their reporter's current period
		if err := k.Keeper.SettleReporter(ctx, selection.Reporter); err != nil {
			return nil, err
		}
	}

	shares, err := k.Keeper.SelectorTips.Get(ctx, selectorAddr)
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
	newShares, err := k.Keeper.stakingKeeper.Delegate(ctx, selectorAddr, amtToDelegate, val.Status, val, false)
	if err != nil {
		return nil, err
	}

	// isolate decimals from shares
	remainder := shares.Sub(shares.TruncateDec())
	if remainder.IsZero() {
		err = k.Keeper.SelectorTips.Remove(ctx, selectorAddr)
		if err != nil {
			return nil, err
		}
	} else {
		err = k.Keeper.SelectorTips.Set(ctx, selectorAddr, remainder)
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
			sdk.NewAttribute("shares", newShares.String()),
		),
	})
	// allow for people to track the amount they have withdrawn based on their address
	telemetry.IncrCounterWithLabels([]string{"withdrawn_amount_tracker"}, float32(amtToDelegate.Int64()), []metrics.Label{{Name: "chain_id", Value: ctx.ChainID()}, {Name: "reporter", Value: hex.EncodeToString(selectorAddr.Bytes())}})
	return &types.MsgWithdrawTipResponse{}, nil
}

func validateWithdrawTip(msg *types.MsgWithdrawTip) (selector sdk.AccAddress, err error) {
	selector, err = sdk.AccAddressFromBech32(msg.SelectorAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid selector address (%s)", err)
	}
	return selector, nil
}

func (k msgServer) EditReporter(goCtx context.Context, msg *types.MsgEditReporter) (*types.MsgEditReporterResponse, error) {
	reporterAddr, err := validateEditReporter(msg)
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(goCtx)

	params, err := k.Keeper.Params.Get(goCtx)
	if err != nil {
		return nil, err
	}

	// reporter commission rate must be between 0 and 1
	if msg.CommissionRate.GT(math.LegacyNewDec(1)) || msg.CommissionRate.LT(params.MinCommissionRate) {
		return nil, errors.New("commission rate must be between 0 and 1 (e.g, 0.50 = 50%)")
	}

	reporter, err := k.Keeper.Reporter(goCtx, reporterAddr)
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

	err = k.Keeper.Reporters.Set(goCtx, reporterAddr.Bytes(), reporter)
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

func validateEditReporter(msg *types.MsgEditReporter) (reporter sdk.AccAddress, err error) {
	reporter, err = sdk.AccAddressFromBech32(msg.ReporterAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid reporter address (%s)", err)
	}

	// check that mintokensrequired is positive
	if msg.MinTokensRequired.LTE(math.ZeroInt()) {
		return nil, errors.New("MinTokensRequired must be positive (%s)")
	}

	// check that moniker is not empty
	if msg.Moniker == "" {
		return nil, errors.New("moniker cannot be empty")
	}

	return reporter, nil
}
