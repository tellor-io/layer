package ante

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	MaxNestedMsgCount = 7
	// ActiveSetDelegationCheckGas makes active-set delegation expansion visible to gas accounting instead of allowing free ante-time scans.
	ActiveSetDelegationCheckGas   = storetypes.Gas(1_000_000)
	activeSetDelegationGasMessage = "active set delegation stake share check"
)

// TrackStakeChangesDecorator is an AnteDecorator that checks if the transaction is going to change stake by more than 5% and disallows the transaction to enter the mempool or be executed if so
type TrackStakeChangesDecorator struct {
	reporterKeeper keeper.Keeper
	stakingKeeper  types.StakingKeeper
}

type delegatorAddressKey string

func newDelegatorAddressKey(addr sdk.AccAddress) delegatorAddressKey {
	return delegatorAddressKey(addr.String())
}

func (k delegatorAddressKey) address() (sdk.AccAddress, error) {
	return sdk.AccAddressFromBech32(string(k))
}

type validatorAddressKey string

func newValidatorAddressKey(addr sdk.ValAddress) validatorAddressKey {
	return validatorAddressKey(addr.String())
}

type reporterAddressKey string

func newReporterAddressKey(addr sdk.AccAddress) reporterAddressKey {
	return reporterAddressKey(addr.String())
}

func (k reporterAddressKey) address() (sdk.AccAddress, error) {
	return sdk.AccAddressFromBech32(string(k))
}

type stakeChangeTracker struct {
	totalBondedDelta     math.Int
	delegatorBondedDelta map[delegatorAddressKey]math.LegacyDec
	delegationShareDelta map[validatorAddressKey]map[delegatorAddressKey]math.LegacyDec
	validatorProjections map[validatorAddressKey]prospectiveValidator
	// pendingValidators holds MsgCreateValidator candidates because they do not
	// exist in staking keeper state yet while ante is running.
	pendingValidators map[validatorAddressKey]prospectiveValidator
	// activeSetDelta is true when a tx can change which validators are bonded.
	activeSetDelta bool
	// selectionChanges records selectors whose selected reporter changes within
	// this tx (CreateReporter/SelectReporter/SwitchReporter), so the reporter
	// power cap books their full stake against the new reporter and later
	// staking deltas in the same tx attribute to the right reporter.
	selectionChanges             map[delegatorAddressKey]reporterAddressKey
	hasDirectStakeAcquisition    bool
	hasProjectedPowerAcquisition bool
}

type prospectiveValidator struct {
	addr       sdk.ValAddress
	validator  stakingtypes.Validator
	postTokens math.Int
	postShares math.LegacyDec
	pending    bool
	// preTxActive: bonded and not jailed before the tx. Jail can leave IsBonded()
	// true while dropping the power index; the validator cap treats that as 0.
	preTxActive bool
	// jailCleared: MsgUnjail; validator-cap only, not 5%/delegator/reporter paths.
	jailCleared bool
}

func NewTrackStakeChangesDecorator(rk keeper.Keeper, sk types.StakingKeeper) TrackStakeChangesDecorator {
	return TrackStakeChangesDecorator{
		reporterKeeper: rk,
		stakingKeeper:  sk,
	}
}

func newStakeChangeTracker() *stakeChangeTracker {
	return &stakeChangeTracker{
		totalBondedDelta:     math.ZeroInt(),
		delegatorBondedDelta: make(map[delegatorAddressKey]math.LegacyDec),
		delegationShareDelta: make(map[validatorAddressKey]map[delegatorAddressKey]math.LegacyDec),
		validatorProjections: make(map[validatorAddressKey]prospectiveValidator),
		pendingValidators:    make(map[validatorAddressKey]prospectiveValidator),
		selectionChanges:     make(map[delegatorAddressKey]reporterAddressKey),
	}
}

func decFromMap[K comparable](values map[K]math.LegacyDec, key K) math.LegacyDec {
	value, ok := values[key]
	if !ok {
		return math.LegacyZeroDec()
	}
	return value
}

func addDec[K comparable](values map[K]math.LegacyDec, key K, amount math.LegacyDec) {
	if amount.IsZero() {
		return
	}
	values[key] = decFromMap(values, key).Add(amount)
}

// sortedKeys keeps validation order deterministic when projections are backed
// by maps. The returned order must not affect consensus results or error order.
func sortedKeys[K ~string, V any](values map[K]V) []K {
	keys := make([]K, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})
	return keys
}

func (t *stakeChangeTracker) add(delegator sdk.AccAddress, amount math.Int) {
	if amount.IsZero() {
		return
	}
	t.totalBondedDelta = t.totalBondedDelta.Add(amount)
	t.addDelegatorDelta(delegator, amount.ToLegacyDec())
}

func (t *stakeChangeTracker) addTotalDelta(amount math.Int) {
	if amount.IsZero() {
		return
	}
	t.totalBondedDelta = t.totalBondedDelta.Add(amount)
}

func (t *stakeChangeTracker) addDelegatorDelta(delegator sdk.AccAddress, amount math.LegacyDec) {
	if amount.IsZero() {
		return
	}
	if delegator == nil {
		return
	}
	addDec(t.delegatorBondedDelta, newDelegatorAddressKey(delegator), amount)
}

func (t *stakeChangeTracker) markActiveSetDelta(activeSetDelta bool) {
	if activeSetDelta {
		t.activeSetDelta = true
	}
}

func (t *stakeChangeTracker) jailCleared() bool {
	for _, validator := range t.validatorProjections {
		if validator.jailCleared {
			return true
		}
	}
	return false
}

func (t *stakeChangeTracker) setSelection(selector, reporter sdk.AccAddress) {
	if selector == nil || reporter == nil {
		return
	}
	t.selectionChanges[newDelegatorAddressKey(selector)] = newReporterAddressKey(reporter)
}

func (t *stakeChangeTracker) addDelegationShareDelta(validator sdk.ValAddress, delegator sdk.AccAddress, shares math.LegacyDec) {
	if shares.IsZero() {
		return
	}
	validatorKey := newValidatorAddressKey(validator)
	if _, ok := t.delegationShareDelta[validatorKey]; !ok {
		t.delegationShareDelta[validatorKey] = make(map[delegatorAddressKey]math.LegacyDec)
	}
	addDec(t.delegationShareDelta[validatorKey], newDelegatorAddressKey(delegator), shares)
}

// addPendingValidator records a MsgCreateValidator candidate that does not exist in staking keeper state yet, so later same-tx messages can still project it.
func (t *stakeChangeTracker) addPendingValidator(validator sdk.ValAddress, amount math.Int) {
	validatorKey := newValidatorAddressKey(validator)
	pending := prospectiveValidator{
		addr: validator,
		validator: stakingtypes.Validator{
			OperatorAddress:   validator.String(),
			Status:            stakingtypes.Unbonded,
			Tokens:            math.ZeroInt(),
			DelegatorShares:   math.LegacyZeroDec(),
			MinSelfDelegation: math.OneInt(),
		},
		postTokens: amount,
		postShares: amount.ToLegacyDec(),
		pending:    true,
	}
	t.pendingValidators[validatorKey] = pending
	t.validatorProjections[validatorKey] = pending
	t.activeSetDelta = true
}

// setProjectedValidator stores the latest post-message validator state for this tx. Later messages read this instead of stale keeper state.
func (t *stakeChangeTracker) setProjectedValidator(validator prospectiveValidator) {
	validatorKey := newValidatorAddressKey(validator.addr)
	t.validatorProjections[validatorKey] = validator
	if validator.pending {
		t.pendingValidators[validatorKey] = validator
	}
}

// projectedValidator returns the current tx projection for a validator, loading keeper state only the first time an existing validator is touched.
func (t *stakeChangeTracker) projectedValidator(ctx sdk.Context, stakingKeeper types.StakingKeeper, valAddr sdk.ValAddress) (prospectiveValidator, error) {
	validatorKey := newValidatorAddressKey(valAddr)
	if validator, ok := t.validatorProjections[validatorKey]; ok {
		return validator, nil
	}
	validator, err := stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return prospectiveValidator{}, err
	}
	projected := prospectiveValidator{
		addr:        valAddr,
		validator:   validator,
		postTokens:  validator.Tokens,
		postShares:  validator.DelegatorShares,
		preTxActive: validator.IsBonded() && !validator.IsJailed(),
	}
	t.validatorProjections[validatorKey] = projected
	return projected, nil
}

// postState materializes the validator after all tracked token/share changes so staking's own share conversion helpers can be reused.
func (v prospectiveValidator) postState() stakingtypes.Validator {
	validator := v.validator
	validator.Tokens = v.postTokens
	validator.DelegatorShares = v.postShares
	return validator
}

// withPostState copies staking's updated token/share fields back into the projection while preserving the original bonded status used for delta checks.
func (v prospectiveValidator) withPostState(validator stakingtypes.Validator) prospectiveValidator {
	v.postTokens = validator.Tokens
	v.postShares = validator.DelegatorShares
	return v
}

// implement the AnteDecorator interface
func (t TrackStakeChangesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// check if the message type will change stake by more than 5%
	stakeChanges := newStakeChangeTracker()
	for _, msg := range tx.GetMsgs() {
		if err := t.processMessage(ctx, msg, 1, stakeChanges); err != nil {
			return ctx, err
		}
	}
	// Reject mixing keeper-direct and ante-projected stake acquisition in one tx.
	if stakeChanges.hasDirectStakeAcquisition && stakeChanges.hasProjectedPowerAcquisition {
		return ctx, types.ErrMixedStakeAcquisitionPaths
	}
	// Allows multi-validator genesis
	if ctx.BlockHeight() > 0 {
		if err := t.finalizeStakeChanges(ctx, stakeChanges); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// finalizeStakeChanges runs the stake limits once against the final projected tx state. This avoids false failures for atomic txs that temporarily cross a threshold and then offset before handlers finish.
func (t TrackStakeChangesDecorator) finalizeStakeChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker) error {
	capCtx, err := t.buildStakeCapContext(ctx, stakeChanges)
	if err != nil {
		return err
	}
	if err := t.checkDelegatorStakeShares(ctx, stakeChanges, capCtx); err != nil {
		return err
	}
	if err := t.checkValidatorPowerShares(ctx, stakeChanges, capCtx); err != nil {
		return err
	}
	return t.checkReporterPowerShares(ctx, stakeChanges, capCtx)
}

func (t TrackStakeChangesDecorator) processMessage(ctx sdk.Context, msg sdk.Msg, nestedMsgCount int64, stakeChanges *stakeChangeTracker) error {
	if nestedMsgCount > MaxNestedMsgCount {
		return fmt.Errorf("nested message count exceeds the maximum allowed: Limit is %d", MaxNestedMsgCount)
	}
	switch msg := msg.(type) {
	// if the message is an authz exec, check the inner messages for any stake changes
	case *authz.MsgExec:
		innerMsgs, err := msg.GetMessages()
		if err != nil {
			return err
		}
		for _, innerMsg := range innerMsgs {
			nestedMsgCount++
			if err := t.processMessage(ctx, innerMsg, nestedMsgCount, stakeChanges); err != nil {
				return err
			}
		}
	// if the message is not an authz exec, check if it is a stake change message
	default:
		if err := t.checkStakeChange(ctx, msg, stakeChanges); err != nil {
			return err
		}
	}
	return nil
}

func isDirectStakeAcquisitionMessage(msg sdk.Msg) bool {
	// Keeper-path claims; ante cannot project the eventual amount.
	switch msg.(type) {
	case *types.MsgWithdrawTip, *disputetypes.MsgWithdrawFeeRefund:
		return true
	default:
		return false
	}
}

func isProjectedPowerAcquisitionMessage(msg sdk.Msg) bool {
	// Ante-projected validator/reporter power acquisitions.
	switch msg.(type) {
	case *types.MsgCreateReporter,
		*types.MsgSelectReporter,
		*types.MsgSwitchReporter,
		*stakingtypes.MsgCreateValidator,
		*stakingtypes.MsgDelegate,
		*stakingtypes.MsgBeginRedelegate,
		*stakingtypes.MsgCancelUnbondingDelegation:
		return true
	default:
		return false
	}
}

func (t TrackStakeChangesDecorator) checkStakeChange(ctx sdk.Context, msg sdk.Msg, stakeChanges *stakeChangeTracker) error {
	if isDirectStakeAcquisitionMessage(msg) {
		stakeChanges.hasDirectStakeAcquisition = true
	}
	if isProjectedPowerAcquisitionMessage(msg) {
		stakeChanges.hasProjectedPowerAcquisition = true
	}

	switch msg := msg.(type) {
	case *types.MsgCreateReporter:
		addr, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
		if err != nil {
			return err
		}
		// the creator's own bonded stake becomes the new reporter's power (both
		// the fresh-create and selector-conversion paths)
		stakeChanges.setSelection(addr, addr)
	case *types.MsgSelectReporter:
		selectorAddr, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
		if err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
		if err != nil {
			return err
		}
		stakeChanges.setSelection(selectorAddr, reporterAddr)
	case *types.MsgSwitchReporter:
		selectorAddr, err := sdk.AccAddressFromBech32(msg.SelectorAddress)
		if err != nil {
			return err
		}
		reporterAddr, err := sdk.AccAddressFromBech32(msg.ReporterAddress)
		if err != nil {
			return err
		}
		// a switch already pending to this reporter is a handler no-op and its
		// stake is already booked against the destination's potential stake
		pending, pendingTo, err := t.reporterKeeper.PendingSwitchTarget(ctx, selectorAddr)
		if err != nil {
			return err
		}
		if pending && bytes.Equal(pendingTo, reporterAddr.Bytes()) {
			return nil
		}
		stakeChanges.setSelection(selectorAddr, reporterAddr)
	case *stakingtypes.MsgCreateValidator:
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return err
		}
		delegatorAddr := sdk.AccAddress(valAddr)
		stakeChanges.addPendingValidator(valAddr, msg.Value.Amount)
		stakeChanges.addDelegationShareDelta(valAddr, delegatorAddr, msg.Value.Amount.ToLegacyDec())
	case *stakingtypes.MsgDelegate:
		isAllowed, err := t.checkAmountOfDelegationsByAddressDoesNotExceedMax(ctx, msg)
		if err != nil {
			return err
		}
		if !isAllowed {
			return types.ErrExceedsMaxDelegations
		}
		delegatorAddr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return err
		}
		validator, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, valAddr)
		if err != nil {
			return err
		}
		postValidator, issuedShares := validator.postState().AddTokensFromDel(msg.Amount.Amount)
		stakeChanges.setProjectedValidator(validator.withPostState(postValidator))
		stakeChanges.addDelegationShareDelta(valAddr, delegatorAddr, issuedShares)
		stakeChanges.markActiveSetDelta(!validator.validator.IsBonded())
		if validator.validator.IsBonded() {
			stakeChanges.add(delegatorAddr, msg.Amount.Amount)
		}
	case *stakingtypes.MsgBeginRedelegate:
		isAllowed, err := t.checkAmountOfDelegationsByAddressDoesNotExceedMax(ctx, msg)
		if err != nil {
			return err
		}
		if !isAllowed {
			return types.ErrExceedsMaxDelegations
		}
		delegatorAddr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		srcValAddr, err := sdk.ValAddressFromBech32(msg.ValidatorSrcAddress)
		if err != nil {
			return err
		}
		dstValAddr, err := sdk.ValAddressFromBech32(msg.ValidatorDstAddress)
		if err != nil {
			return err
		}

		sourceVal, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, srcValAddr)
		if err != nil {
			return err
		}
		destVal, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, dstValAddr)
		if err != nil {
			return err
		}
		shares, err := t.projectedUnbondShares(ctx, stakeChanges, delegatorAddr, sourceVal, msg.Amount.Amount)
		if err != nil {
			return err
		}
		sourcePost, returnAmount := sourceVal.postState().RemoveDelShares(shares)
		destPost, issuedShares := destVal.postState().AddTokensFromDel(returnAmount)
		stakeChanges.setProjectedValidator(sourceVal.withPostState(sourcePost))
		stakeChanges.setProjectedValidator(destVal.withPostState(destPost))
		stakeChanges.addDelegationShareDelta(srcValAddr, delegatorAddr, shares.Neg())
		stakeChanges.addDelegationShareDelta(dstValAddr, delegatorAddr, issuedShares)
		stakeChanges.markActiveSetDelta(true)
		stakeChanges.markActiveSetDelta(!destVal.validator.IsBonded())
		switch {
		case sourceVal.validator.IsBonded() && !destVal.validator.IsBonded():
			stakeChanges.add(delegatorAddr, returnAmount.Neg())
		case !sourceVal.validator.IsBonded() && destVal.validator.IsBonded():
			stakeChanges.add(delegatorAddr, returnAmount)
		}
	case *stakingtypes.MsgCancelUnbondingDelegation:
		delegatorAddr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return err
		}
		validator, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, valAddr)
		if err != nil {
			return err
		}
		postValidator, issuedShares := validator.postState().AddTokensFromDel(msg.Amount.Amount)
		stakeChanges.setProjectedValidator(validator.withPostState(postValidator))
		stakeChanges.addDelegationShareDelta(valAddr, delegatorAddr, issuedShares)
		stakeChanges.markActiveSetDelta(!validator.validator.IsBonded())
		if validator.validator.IsBonded() {
			stakeChanges.add(delegatorAddr, msg.Amount.Amount)
		}
	case *stakingtypes.MsgUndelegate:
		delegatorAddr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return err
		}
		validator, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, valAddr)
		if err != nil {
			return err
		}
		shares, err := t.projectedUnbondShares(ctx, stakeChanges, delegatorAddr, validator, msg.Amount.Amount)
		if err != nil {
			return err
		}
		postValidator, returnAmount := validator.postState().RemoveDelShares(shares)
		stakeChanges.setProjectedValidator(validator.withPostState(postValidator))
		stakeChanges.addDelegationShareDelta(valAddr, delegatorAddr, shares.Neg())
		stakeChanges.markActiveSetDelta(true)
		if validator.validator.IsBonded() {
			stakeChanges.add(delegatorAddr, returnAmount.Neg())
		}
	case *slashingtypes.MsgUnjail:
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddr)
		if err != nil {
			return err
		}
		validator, err := stakeChanges.projectedValidator(ctx, t.stakingKeeper, valAddr)
		if err != nil {
			return err
		}
		// pure unjail: project active-set re-entry for the validator cap only
		validator.validator.Jailed = false
		validator.jailCleared = true
		stakeChanges.setProjectedValidator(validator)
	default:
		return nil
	}
	return nil
}

// projectedUnbondShares mirrors staking's ValidateUnbondAmount logic against
// projected validator/delegation state, including rounding and full-withdraw
// share capping behavior.
func (t TrackStakeChangesDecorator) projectedUnbondShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, delegator sdk.AccAddress, validator prospectiveValidator, amount math.Int) (math.LegacyDec, error) {
	postValidator := validator.postState()
	shares, err := postValidator.SharesFromTokens(amount)
	if err != nil {
		return math.LegacyDec{}, err
	}
	sharesTruncated, err := postValidator.SharesFromTokensTruncated(amount)
	if err != nil {
		return math.LegacyDec{}, err
	}
	delegationShares, err := t.projectedDelegationShares(ctx, stakeChanges, delegator, validator.addr)
	if err != nil {
		return math.LegacyDec{}, err
	}
	if sharesTruncated.GT(delegationShares) {
		return math.LegacyDec{}, fmt.Errorf("invalid shares amount")
	}
	if shares.GT(delegationShares) {
		return delegationShares, nil
	}
	return shares, nil
}

// projectedDelegationShares returns the delegator's shares after earlier
// messages in this tx, without mutating staking state.
func (t TrackStakeChangesDecorator) projectedDelegationShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, delegator sdk.AccAddress, validator sdk.ValAddress) (math.LegacyDec, error) {
	shares := math.LegacyZeroDec()
	delegation, err := t.stakingKeeper.GetDelegation(ctx, delegator, validator)
	if err != nil && !errors.Is(err, stakingtypes.ErrNoDelegation) {
		return math.LegacyDec{}, err
	}
	if err == nil {
		shares = delegation.Shares
	}
	validatorKey := newValidatorAddressKey(validator)
	delegatorKey := newDelegatorAddressKey(delegator)
	shares = shares.Add(decFromMap(stakeChanges.delegationShareDelta[validatorKey], delegatorKey))
	if shares.IsNegative() {
		return math.LegacyDec{}, fmt.Errorf("projected delegation shares cannot be negative")
	}
	return shares, nil
}

// checkTotalStakeChange enforces the 5% total bonded-token movement limit using
// the final projected bonded delta for the whole tx.
func (t TrackStakeChangesDecorator) checkTotalStakeChange(ctx sdk.Context, totalBondedDelta math.Int) error {
	if totalBondedDelta.IsZero() {
		return nil
	}
	lastupdated, err := t.reporterKeeper.Tracker.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	currentAmount, err := t.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	changeAmt := currentAmount.Add(totalBondedDelta)
	if totalBondedDelta.IsNegative() {
		allowedLowerBound := lastupdated.Amount.Sub(lastupdated.Amount.QuoRaw(20))
		if changeAmt.LT(allowedLowerBound) {
			return errors.New("total stake decrease exceeds the allowed 5% threshold within a twelve-hour period")
		}
		return nil
	}
	allowedUpperBound := lastupdated.Amount.Add(lastupdated.Amount.QuoRaw(20))
	if changeAmt.GT(allowedUpperBound) {
		return errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
	}
	return nil
}

// checkDelegatorStakeShares enforces the 30% bonded-stake cap only for
// delegators whose projected bonded stake increases.
func (t TrackStakeChangesDecorator) checkDelegatorStakeShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, capCtx stakeCapContext) error {
	if stakeChanges == nil || len(stakeChanges.delegatorBondedDelta) == 0 {
		return nil
	}
	totalBondedAfter := capCtx.totalBondedAfterDelegator()
	if !totalBondedAfter.IsPositive() {
		return nil
	}
	for _, delegatorKey := range sortedKeys(stakeChanges.delegatorBondedDelta) {
		delta := stakeChanges.delegatorBondedDelta[delegatorKey]
		if !delta.IsPositive() {
			continue
		}
		delegator, err := delegatorKey.address()
		if err != nil {
			return err
		}
		currentDelegatorBonded, err := t.delegatorBondedTokens(ctx, delegator)
		if err != nil {
			return err
		}
		delegatorBondedAfter := currentDelegatorBonded.Add(delta)
		if types.ExceedsDelegatorStakeShare(delegatorBondedAfter, totalBondedAfter) {
			return types.ErrExceedsMaxStakeShare
		}
	}
	return nil
}

// delegatorBondedTokens sums a delegator's currently bonded stake across all
// bonded validators using staking's share-to-token conversion.
func (t TrackStakeChangesDecorator) delegatorBondedTokens(ctx sdk.Context, delegator sdk.AccAddress) (math.LegacyDec, error) {
	tokens := math.LegacyZeroDec()
	var iterError error
	err := t.stakingKeeper.IterateDelegatorDelegations(ctx, delegator, func(delegation stakingtypes.Delegation) (stop bool) {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			iterError = err
			return true
		}
		val, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			iterError = err
			return true
		}
		if val.IsBonded() {
			tokens = tokens.Add(val.TokensFromShares(delegation.Shares))
		}
		return false
	})
	if err != nil {
		return math.LegacyDec{}, err
	}
	return tokens, iterError
}

// checkReporterPowerShares enforces the reporter power cap: no reporter's
// projected potential stake may reach the max_reporter_power_share fraction of
// projected total bonded stake. Only reporters gaining stake in this tx are
// checked; decreases are never blocked, so an over-cap reporter can always
// shed stake.
func (t TrackStakeChangesDecorator) checkReporterPowerShares(ctx sdk.Context, stakeChanges *stakeChangeTracker, capCtx stakeCapContext) error {
	if stakeChanges == nil || (len(stakeChanges.selectionChanges) == 0 && len(stakeChanges.delegatorBondedDelta) == 0) {
		return nil
	}
	params, err := t.reporterKeeper.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	maxShare := params.MaxReporterPowerShare
	// nil/zero is pre-migration state and shares >= 1 are explicitly disabled;
	// both must not be read as "cap everything at zero"
	if !types.PowerShareEnabled(maxShare) {
		return nil
	}

	reporterAdditions := make(map[reporterAddressKey]math.LegacyDec)
	// selectors changing reporter bring their whole projected bonded stake to
	// the destination reporter
	for _, selectorKey := range sortedKeys(stakeChanges.selectionChanges) {
		selector, err := selectorKey.address()
		if err != nil {
			return err
		}
		bonded, err := t.delegatorBondedTokens(ctx, selector)
		if err != nil {
			return err
		}
		contribution := bonded.Add(decFromMap(stakeChanges.delegatorBondedDelta, selectorKey))
		if contribution.IsPositive() {
			addDec(reporterAdditions, stakeChanges.selectionChanges[selectorKey], contribution)
		}
	}
	// stake increases by existing selectors attribute to their selected reporter
	for _, delegatorKey := range sortedKeys(stakeChanges.delegatorBondedDelta) {
		if _, changed := stakeChanges.selectionChanges[delegatorKey]; changed {
			continue // already counted above with the selector's full stake
		}
		delta := stakeChanges.delegatorBondedDelta[delegatorKey]
		if !delta.IsPositive() {
			continue
		}
		delegator, err := delegatorKey.address()
		if err != nil {
			return err
		}
		reporter, found, err := t.selectedReporter(ctx, delegator)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		addDec(reporterAdditions, newReporterAddressKey(reporter), delta)
	}
	if len(reporterAdditions) == 0 {
		return nil
	}

	totalBondedAfter := capCtx.totalBondedAfterDelegator()
	if !totalBondedAfter.IsPositive() {
		return nil
	}
	for _, reporterKey := range sortedKeys(reporterAdditions) {
		reporter, err := reporterKey.address()
		if err != nil {
			return err
		}
		potential, err := t.reporterKeeper.ReporterPotentialStake(ctx, reporter)
		if err != nil {
			return err
		}
		if types.ExceedsReporterPowerShare(potential.ToLegacyDec(), reporterAdditions[reporterKey], totalBondedAfter, maxShare) {
			return errorsmod.Wrapf(types.ErrExceedsMaxReporterPower, "reporter %s", reporter.String())
		}
	}
	return nil
}

// selectedReporter resolves the reporter a delegator's stake counts toward: the
// pending switch destination when one is scheduled, otherwise the stored
// selection. Returns found=false for delegators who are not selectors.
func (t TrackStakeChangesDecorator) selectedReporter(ctx sdk.Context, delegator sdk.AccAddress) (sdk.AccAddress, bool, error) {
	selection, err := t.reporterKeeper.GetSelector(ctx, delegator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	pending, to, err := t.reporterKeeper.PendingSwitchTarget(ctx, delegator)
	if err != nil {
		return nil, false, err
	}
	if pending {
		return sdk.AccAddress(to), true, nil
	}
	return sdk.AccAddress(selection.Reporter), true, nil
}

func (t TrackStakeChangesDecorator) checkAmountOfDelegationsByAddressDoesNotExceedMax(ctx sdk.Context, msg sdk.Msg) (bool, error) {
	params, err := t.reporterKeeper.Params.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return true, nil
		}
		return false, err
	}
	switch msg := msg.(type) {
	case *stakingtypes.MsgDelegate:
		addr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
		delegations, err := t.stakingKeeper.GetAllDelegatorDelegations(ctx, addr)
		if err != nil {
			return false, err
		}

		// Check to ensure that the number of delegations does not exceed 10
		if len(delegations) == int(params.MaxNumOfDelegations) {
			return false, nil
		}
		return true, nil
	case *stakingtypes.MsgBeginRedelegate:
		addr := sdk.MustAccAddressFromBech32(msg.DelegatorAddress)
		delegations, err := t.stakingKeeper.GetAllDelegatorDelegations(ctx, addr)
		if err != nil {
			return false, err
		}

		// Check to ensure that the number of delegations does not exceed 10
		if len(delegations) == int(params.MaxNumOfDelegations) {
			for i := 0; i < int(params.MaxNumOfDelegations); i++ {
				if strings.EqualFold(delegations[i].ValidatorAddress, msg.ValidatorSrcAddress) {
					if msg.Amount.Amount.Equal(delegations[i].Shares.TruncateInt()) {
						return true, nil
					} else {
						return false, nil
					}
				}
			}
		}
		return true, nil
	default:
		return true, nil
	}
}
