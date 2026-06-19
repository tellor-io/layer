package ante

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
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
	selectionChanges map[delegatorAddressKey]reporterAddressKey
}

type prospectiveValidator struct {
	addr       sdk.ValAddress
	validator  stakingtypes.Validator
	postTokens math.Int
	postShares math.LegacyDec
	pending    bool
}

type activeSetChanges struct {
	entering []prospectiveValidator
	leaving  []prospectiveValidator
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
		addr:       valAddr,
		validator:  validator,
		postTokens: validator.Tokens,
		postShares: validator.DelegatorShares,
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
	if stakeChanges.activeSetDelta {
		if err := t.applyProspectiveBondedValidatorChanges(ctx, stakeChanges); err != nil {
			return err
		}
	} else {
		if err := t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta); err != nil {
			return err
		}
	}
	if err := t.checkDelegatorStakeShares(ctx, stakeChanges); err != nil {
		return err
	}
	return t.checkReporterPowerShares(ctx, stakeChanges)
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

func (t TrackStakeChangesDecorator) checkStakeChange(ctx sdk.Context, msg sdk.Msg, stakeChanges *stakeChangeTracker) error {
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

// applyProspectiveBondedValidatorChanges accounts for validators entering or
// leaving the active set after tx handlers run, then folds those stake changes
// into the final total and per-delegator checks.
func (t TrackStakeChangesDecorator) applyProspectiveBondedValidatorChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker) error {
	if stakeChanges == nil || !stakeChanges.activeSetDelta {
		return nil
	}
	activeSetChanges, err := t.prospectiveActiveSetChanges(ctx, stakeChanges)
	if err != nil {
		return err
	}
	if len(activeSetChanges.entering) == 0 && len(activeSetChanges.leaving) == 0 {
		return t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta)
	}

	// Replacement changes include both sides: entrants add their post-change
	// stake, while leavers remove the stake they would still have after the tx.
	prospectiveBondedDelta := math.ZeroInt()
	for _, validator := range activeSetChanges.entering {
		prospectiveBondedDelta = prospectiveBondedDelta.Add(validator.postTokens)
	}
	for _, validator := range activeSetChanges.leaving {
		prospectiveBondedDelta = prospectiveBondedDelta.Sub(validator.postTokens)
	}
	stakeChanges.addTotalDelta(prospectiveBondedDelta)
	if err := t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta); err != nil {
		return err
	}
	for _, validator := range activeSetChanges.entering {
		if err := t.addActiveSetValidatorDelegatorDeltas(ctx, stakeChanges, validator, math.LegacyOneDec()); err != nil {
			return err
		}
	}
	for _, validator := range activeSetChanges.leaving {
		if err := t.addActiveSetValidatorDelegatorDeltas(ctx, stakeChanges, validator, math.LegacyNewDec(-1)); err != nil {
			return err
		}
	}
	return nil
}

// prospectiveActiveSetChanges projects staking's next active set from the
// current top validators plus validators touched by this tx. It keeps the scan
// bounded and sorts candidates to preserve deterministic behavior.
func (t TrackStakeChangesDecorator) prospectiveActiveSetChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker) (activeSetChanges, error) {
	maxValidators, err := t.stakingKeeper.MaxValidators(ctx)
	if err != nil {
		return activeSetChanges{}, err
	}
	if maxValidators == 0 {
		return activeSetChanges{}, nil
	}
	powerReduction := t.stakingKeeper.PowerReduction(ctx)
	validators := make(map[validatorAddressKey]prospectiveValidator)
	iterator, err := t.stakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		return activeSetChanges{}, err
	}
	defer iterator.Close()

	// Scan the current top set plus enough replacement candidates to cover each
	// validator touched by this transaction.
	scanLimit := int(maxValidators) + len(stakeChanges.validatorProjections)
	for count := 0; iterator.Valid() && count < scanLimit; iterator.Next() {
		valAddr := sdk.ValAddress(iterator.Value())
		validatorKey := newValidatorAddressKey(valAddr)
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		validator, ok := stakeChanges.validatorProjections[validatorKey]
		if !ok {
			current, err := t.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return activeSetChanges{}, err
			}
			validator = prospectiveValidator{
				addr:       valAddr,
				validator:  current,
				postTokens: current.Tokens,
				postShares: current.DelegatorShares,
			}
		}
		validators[validatorKey] = validator
		count++
	}

	// Add validators touched by this tx that were not present in the power
	// index scan, including validators created by MsgCreateValidator.
	for _, validatorKey := range sortedKeys(stakeChanges.validatorProjections) {
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		validators[validatorKey] = stakeChanges.validatorProjections[validatorKey]
	}

	ordered := make([]prospectiveValidator, 0, len(validators))
	for _, validatorKey := range sortedKeys(validators) {
		validator := validators[validatorKey]
		if sdk.TokensToConsensusPower(validator.postTokens, powerReduction) == 0 {
			continue
		}
		ordered = append(ordered, validator)
	}
	// Match staking's active-set ranking: consensus power first, then operator
	// address to make ties deterministic.
	sort.Slice(ordered, func(i, j int) bool {
		iPower := sdk.TokensToConsensusPower(ordered[i].postTokens, powerReduction)
		jPower := sdk.TokensToConsensusPower(ordered[j].postTokens, powerReduction)
		if iPower == jPower {
			return bytes.Compare(ordered[i].addr, ordered[j].addr) < 0
		}
		return iPower > jPower
	})

	limit := int(maxValidators)
	if len(ordered) < limit {
		limit = len(ordered)
	}
	nextSet := make(map[validatorAddressKey]struct{}, limit)
	for _, validator := range ordered[:limit] {
		nextSet[newValidatorAddressKey(validator.addr)] = struct{}{}
	}

	changes := activeSetChanges{}
	for _, validatorKey := range sortedKeys(validators) {
		validator := validators[validatorKey]
		_, inNextSet := nextSet[validatorKey]
		switch {
		case inNextSet && !validator.validator.IsBonded():
			changes.entering = append(changes.entering, validator)
		case !inNextSet && validator.validator.IsBonded():
			changes.leaving = append(changes.leaving, validator)
		}
	}
	return changes, nil
}

// addActiveSetValidatorDelegatorDeltas adds the per-delegator bonded stake that
// appears or disappears when a validator enters or leaves the active set.
func (t TrackStakeChangesDecorator) addActiveSetValidatorDelegatorDeltas(ctx sdk.Context, stakeChanges *stakeChangeTracker, validator prospectiveValidator, sign math.LegacyDec) error {
	delegatorShares := make(map[delegatorAddressKey]math.LegacyDec)
	validatorKey := newValidatorAddressKey(validator.addr)
	// Pending validators have no stored delegations yet; their self-delegation
	// is already represented in delegationShareDelta.
	if _, pending := stakeChanges.pendingValidators[validatorKey]; !pending {
		delegations, err := t.stakingKeeper.GetValidatorDelegations(ctx, validator.addr)
		if err != nil {
			return err
		}
		for _, delegation := range delegations {
			ctx.GasMeter().ConsumeGas(ActiveSetDelegationCheckGas, activeSetDelegationGasMessage)
			delegator, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			if err != nil {
				return err
			}
			addDec(delegatorShares, newDelegatorAddressKey(delegator), delegation.Shares)
		}
	}
	for _, delegatorKey := range sortedKeys(stakeChanges.delegationShareDelta[validatorKey]) {
		delta := stakeChanges.delegationShareDelta[validatorKey][delegatorKey]
		addDec(delegatorShares, delegatorKey, delta)
	}
	postValidator := validator.postState()
	for _, delegatorKey := range sortedKeys(delegatorShares) {
		shares := delegatorShares[delegatorKey]
		if shares.IsPositive() {
			delegator, err := delegatorKey.address()
			if err != nil {
				return err
			}
			amount := postValidator.TokensFromShares(shares)
			stakeChanges.addDelegatorDelta(delegator, amount.Mul(sign))
		}
	}
	return nil
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
func (t TrackStakeChangesDecorator) checkDelegatorStakeShares(ctx sdk.Context, stakeChanges *stakeChangeTracker) error {
	if stakeChanges == nil || len(stakeChanges.delegatorBondedDelta) == 0 {
		return nil
	}
	currentTotalBonded, err := t.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	totalBondedAfter := currentTotalBonded.Add(stakeChanges.totalBondedDelta)
	if !totalBondedAfter.IsPositive() {
		return nil
	}
	totalBondedAfterDec := totalBondedAfter.ToLegacyDec()
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
		if delegatorBondedAfter.MulInt64(10).GT(totalBondedAfterDec.MulInt64(3)) {
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
func (t TrackStakeChangesDecorator) checkReporterPowerShares(ctx sdk.Context, stakeChanges *stakeChangeTracker) error {
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
	if maxShare.IsNil() || !maxShare.IsPositive() || maxShare.GTE(math.LegacyOneDec()) {
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

	currentTotalBonded, err := t.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	totalBondedAfter := currentTotalBonded.Add(stakeChanges.totalBondedDelta)
	if !totalBondedAfter.IsPositive() {
		return nil
	}
	maxAllowed := maxShare.MulInt(totalBondedAfter)
	for _, reporterKey := range sortedKeys(reporterAdditions) {
		reporter, err := reporterKey.address()
		if err != nil {
			return err
		}
		potential, err := t.reporterKeeper.ReporterPotentialStake(ctx, reporter)
		if err != nil {
			return err
		}
		if potential.ToLegacyDec().Add(reporterAdditions[reporterKey]).GTE(maxAllowed) {
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
