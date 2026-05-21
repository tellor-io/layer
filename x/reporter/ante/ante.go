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
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	MaxNestedMsgCount = 7
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

func (k validatorAddressKey) address() (sdk.ValAddress, error) {
	return sdk.ValAddressFromBech32(string(k))
}

type stakeChangeTracker struct {
	totalBondedDelta     math.Int
	delegatorBondedDelta map[delegatorAddressKey]math.LegacyDec
	// validatorTokenDelta accumulates per-validator token changes from the tx so
	// ante can model the post-tx active set before staking handlers run.
	validatorTokenDelta map[validatorAddressKey]math.Int
	validatorDeltas     map[validatorAddressKey]map[delegatorAddressKey]math.LegacyDec
	// pendingValidators holds MsgCreateValidator candidates because they do not
	// exist in staking keeper state yet while ante is running.
	pendingValidators map[validatorAddressKey]prospectiveValidator
	// activeSetDelta is true when a tx can change which validators are bonded.
	activeSetDelta bool
}

// pendingValidatorDelta captures one staking message's token movement for a
// validator. amount is signed relative to that validator's token balance.
type pendingValidatorDelta struct {
	validator      sdk.ValAddress
	delegator      sdk.AccAddress
	amount         math.Int
	activeSetDelta bool
}

type prospectiveValidator struct {
	addr       sdk.ValAddress
	validator  stakingtypes.Validator
	postTokens math.Int
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
		validatorTokenDelta:  make(map[validatorAddressKey]math.Int),
		validatorDeltas:      make(map[validatorAddressKey]map[delegatorAddressKey]math.LegacyDec),
		pendingValidators:    make(map[validatorAddressKey]prospectiveValidator),
	}
}

func intFromMap[K comparable](values map[K]math.Int, key K) math.Int {
	value, ok := values[key]
	if !ok {
		return math.ZeroInt()
	}
	return value
}

func addInt[K comparable](values map[K]math.Int, key K, amount math.Int) {
	if amount.IsZero() {
		return
	}
	values[key] = intFromMap(values, key).Add(amount)
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

func (t *stakeChangeTracker) addValidatorDelta(validator sdk.ValAddress, delegator sdk.AccAddress, amount math.Int, activeSetDelta bool) {
	if amount.IsZero() {
		return
	}
	if activeSetDelta {
		t.activeSetDelta = true
	}
	validatorKey := newValidatorAddressKey(validator)
	addInt(t.validatorTokenDelta, validatorKey, amount)
	if delegator == nil {
		return
	}
	if _, ok := t.validatorDeltas[validatorKey]; !ok {
		t.validatorDeltas[validatorKey] = make(map[delegatorAddressKey]math.LegacyDec)
	}
	delegatorKey := newDelegatorAddressKey(delegator)
	addDec(t.validatorDeltas[validatorKey], delegatorKey, amount.ToLegacyDec())
}

func (t *stakeChangeTracker) addPendingValidator(validator sdk.ValAddress, amount math.Int) {
	validatorKey := newValidatorAddressKey(validator)
	t.pendingValidators[validatorKey] = prospectiveValidator{
		addr: validator,
		validator: stakingtypes.Validator{
			OperatorAddress:   validator.String(),
			Status:            stakingtypes.Unbonded,
			Tokens:            math.ZeroInt(),
			DelegatorShares:   math.LegacyZeroDec(),
			MinSelfDelegation: math.OneInt(),
		},
		postTokens: amount,
	}
	t.activeSetDelta = true
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
		if err := t.applyProspectiveBondedValidatorChanges(ctx, stakeChanges); err != nil {
			return ctx, err
		}
		if err := t.checkDelegatorStakeShares(ctx, stakeChanges); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
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
	msgAmount := math.ZeroInt()
	var delegatorAddr sdk.AccAddress
	// validatorDeltas records validator-level effects from this message. These are later merged into stakeChanges so active-set changes can be evaluatedagainst the full transaction, not one message at a time.
	var validatorDeltas []pendingValidatorDelta
	switch msg := msg.(type) {
	case *stakingtypes.MsgCreateValidator:
		valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
		if err != nil {
			return err
		}
		delegatorAddr = sdk.AccAddress(valAddr)
		// A new validator is not stored until the staking handler runs, but its self-delegation can still make it enter the active set after this tx.
		validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
			validator:      valAddr,
			delegator:      delegatorAddr,
			amount:         msg.Value.Amount,
			activeSetDelta: true,
		})
		if stakeChanges != nil {
			stakeChanges.addPendingValidator(valAddr, msg.Value.Amount)
		}
	case *stakingtypes.MsgDelegate:
		isAllowed, err := t.checkAmountOfDelegationsByAddressDoesNotExceedMax(ctx, msg)
		if err != nil {
			return err
		}
		if !isAllowed {
			return types.ErrExceedsMaxDelegations
		}
		delegatorAddr, err = sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		var valAddr sdk.ValAddress
		if addr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err == nil {
			valAddr = addr
		} else {
			return err
		}
		val, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		if val.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
		}
		validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
			validator:      valAddr,
			delegator:      delegatorAddr,
			amount:         msg.Amount.Amount,
			activeSetDelta: !val.IsBonded(),
		})
	case *stakingtypes.MsgBeginRedelegate:
		isAllowed, err := t.checkAmountOfDelegationsByAddressDoesNotExceedMax(ctx, msg)
		if err != nil {
			return err
		}
		if !isAllowed {
			return types.ErrExceedsMaxDelegations
		}
		delegatorAddr, err = sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		// redelegate shouldn't increase the total stake, however if its coming from
		// a validator that is not in the active set, it might be considered as an increase
		// in the active stake. Hence, we need to handle it appropriately.
		var srcValAddr sdk.ValAddress
		if addr, err := sdk.ValAddressFromBech32(msg.ValidatorSrcAddress); err == nil {
			srcValAddr = addr
		} else {
			return err
		}
		var dstValAddr sdk.ValAddress
		if addr, err := sdk.ValAddressFromBech32(msg.ValidatorDstAddress); err == nil {
			dstValAddr = addr
		} else {
			return err
		}

		sourceVal, err := t.stakingKeeper.GetValidator(ctx, srcValAddr)
		if err != nil {
			return err
		}
		destVal, err := t.stakingKeeper.GetValidator(ctx, dstValAddr)
		if err != nil {
			return err
		}

		if sourceVal.Status == stakingtypes.Bonded && destVal.Status != stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount.MulRaw(-1)
			validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
				validator:      srcValAddr,
				delegator:      delegatorAddr,
				amount:         msg.Amount.Amount.Neg(),
				activeSetDelta: true,
			}, pendingValidatorDelta{
				validator:      dstValAddr,
				delegator:      delegatorAddr,
				amount:         msg.Amount.Amount,
				activeSetDelta: true,
			})
		} else if sourceVal.Status == destVal.Status {
			validatorDeltas = append(validatorDeltas,
				pendingValidatorDelta{
					validator:      srcValAddr,
					delegator:      delegatorAddr,
					amount:         msg.Amount.Amount.Neg(),
					activeSetDelta: true,
				},
				pendingValidatorDelta{
					validator:      dstValAddr,
					delegator:      delegatorAddr,
					amount:         msg.Amount.Amount,
					activeSetDelta: !destVal.IsBonded(),
				},
			)
		} else if sourceVal.Status != stakingtypes.Bonded && destVal.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
			validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
				validator:      srcValAddr,
				delegator:      delegatorAddr,
				amount:         msg.Amount.Amount.Neg(),
				activeSetDelta: true,
			}, pendingValidatorDelta{
				validator:      dstValAddr,
				delegator:      delegatorAddr,
				amount:         msg.Amount.Amount,
				activeSetDelta: false,
			})
		}
	case *stakingtypes.MsgCancelUnbondingDelegation:
		var err error
		delegatorAddr, err = sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		var valAddr sdk.ValAddress
		if addr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err == nil {
			valAddr = addr
		} else {
			return err
		}
		val, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		if val.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
		}
		validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
			validator:      valAddr,
			delegator:      delegatorAddr,
			amount:         msg.Amount.Amount,
			activeSetDelta: !val.IsBonded(),
		})
	case *stakingtypes.MsgUndelegate:
		var err error
		delegatorAddr, err = sdk.AccAddressFromBech32(msg.DelegatorAddress)
		if err != nil {
			return err
		}
		var valAddr sdk.ValAddress
		if addr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err == nil {
			valAddr = addr
		} else {
			return err
		}
		val, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return err
		}
		if val.Status == stakingtypes.Bonded {
			// negate the amount since undelegating is removing stake from the chain
			// and to help with the comparison later on
			msgAmount = msg.Amount.Amount.Neg()
		}
		validatorDeltas = append(validatorDeltas, pendingValidatorDelta{
			validator:      valAddr,
			delegator:      delegatorAddr,
			amount:         msg.Amount.Amount.Neg(),
			activeSetDelta: true,
		})
	default:
		return nil
	}

	if msgAmount.IsZero() && len(validatorDeltas) == 0 {
		return nil
	}

	if !msgAmount.IsZero() {
		currentAmount, err := t.stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return err
		}

		// get the total bonded tokens that was set in the last update
		// to compare against the current amount of bonded tokens
		lastupdated, err := t.reporterKeeper.Tracker.Get(ctx)
		if err != nil {
			// for when chain is first started
			if errors.Is(err, collections.ErrNotFound) {
				if stakeChanges != nil {
					for _, delta := range validatorDeltas {
						stakeChanges.addValidatorDelta(delta.validator, delta.delegator, delta.amount, delta.activeSetDelta)
					}
					stakeChanges.add(delegatorAddr, msgAmount)
				}
				return nil
			}
			return err
		}
		changeAmt := currentAmount.Add(msgAmount)
		if stakeChanges != nil {
			changeAmt = currentAmount.Add(stakeChanges.totalBondedDelta).Add(msgAmount)
		}
		if msgAmount.IsNegative() {
			// subtract 5 percent from last updated amount
			allowedLowerBound := lastupdated.Amount.Sub(lastupdated.Amount.QuoRaw(20))
			if changeAmt.LT(allowedLowerBound) {
				return errors.New("total stake decrease exceeds the allowed 5% threshold within a twelve-hour period")
			}
		} else {
			// add 5 percent to last updated amount
			allowedUpperBound := lastupdated.Amount.Add(lastupdated.Amount.QuoRaw(20))
			if changeAmt.GT(allowedUpperBound) {
				return errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
			}
		}
	}
	if stakeChanges != nil {
		for _, delta := range validatorDeltas {
			stakeChanges.addValidatorDelta(delta.validator, delta.delegator, delta.amount, delta.activeSetDelta)
		}
		stakeChanges.add(delegatorAddr, msgAmount)
	}
	return nil
}

func (t TrackStakeChangesDecorator) applyProspectiveBondedValidatorChanges(ctx sdk.Context, stakeChanges *stakeChangeTracker) error {
	if stakeChanges == nil || !stakeChanges.activeSetDelta {
		return nil
	}
	activeSetChanges, err := t.prospectiveActiveSetChanges(ctx, stakeChanges)
	if err != nil {
		return err
	}
	if len(activeSetChanges.entering) == 0 && len(activeSetChanges.leaving) == 0 {
		return nil
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
	if err := t.checkTotalStakeChange(ctx, stakeChanges.totalBondedDelta.Add(prospectiveBondedDelta)); err != nil {
		return err
	}

	stakeChanges.addTotalDelta(prospectiveBondedDelta)
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

	// Start from the power index so an untouched unbonded validator can still enter when a currently bonded validator loses enough stake.
	for ; iterator.Valid(); iterator.Next() {
		valAddr := sdk.ValAddress(iterator.Value())
		validator, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return activeSetChanges{}, err
		}
		validatorDelta := intFromMap(stakeChanges.validatorTokenDelta, newValidatorAddressKey(valAddr))
		postTokens := validator.Tokens.Add(validatorDelta)
		if postTokens.IsNegative() {
			postTokens = math.ZeroInt()
		}
		validators[newValidatorAddressKey(valAddr)] = prospectiveValidator{
			addr:       valAddr,
			validator:  validator,
			postTokens: postTokens,
		}
	}

	// Add validators touched by this tx that were not present in the power
	// index scan, including validators created by MsgCreateValidator.
	for validatorKey, delta := range stakeChanges.validatorTokenDelta {
		if _, ok := validators[validatorKey]; ok {
			continue
		}
		valAddr, err := validatorKey.address()
		if err != nil {
			return activeSetChanges{}, err
		}
		if pending, ok := stakeChanges.pendingValidators[validatorKey]; ok {
			validators[validatorKey] = pending
			continue
		}
		validator, err := t.stakingKeeper.GetValidator(ctx, valAddr)
		if err != nil {
			return activeSetChanges{}, err
		}
		postTokens := validator.Tokens.Add(delta)
		if postTokens.IsNegative() {
			postTokens = math.ZeroInt()
		}
		validators[validatorKey] = prospectiveValidator{
			addr:       valAddr,
			validator:  validator,
			postTokens: postTokens,
		}
	}

	ordered := make([]prospectiveValidator, 0, len(validators))
	for _, validator := range validators {
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
	for validatorKey, validator := range validators {
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

func (t TrackStakeChangesDecorator) addActiveSetValidatorDelegatorDeltas(ctx sdk.Context, stakeChanges *stakeChangeTracker, validator prospectiveValidator, sign math.LegacyDec) error {
	delegatorAmounts := make(map[delegatorAddressKey]math.LegacyDec)
	validatorKey := newValidatorAddressKey(validator.addr)
	// Pending validators have no stored delegations yet; their self-delegation
	// is already represented in validatorDeltas.
	if _, pending := stakeChanges.pendingValidators[validatorKey]; !pending {
		delegations, err := t.stakingKeeper.GetValidatorDelegations(ctx, validator.addr)
		if err != nil {
			return err
		}
		for _, delegation := range delegations {
			delegator, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			if err != nil {
				return err
			}
			amount := validator.validator.TokensFromShares(delegation.Shares)
			addDec(delegatorAmounts, newDelegatorAddressKey(delegator), amount)
		}
	}
	for delegatorKey, delta := range stakeChanges.validatorDeltas[validatorKey] {
		addDec(delegatorAmounts, delegatorKey, delta)
	}
	for delegatorKey, amount := range delegatorAmounts {
		if amount.IsPositive() {
			delegator, err := delegatorKey.address()
			if err != nil {
				return err
			}
			stakeChanges.addDelegatorDelta(delegator, amount.Mul(sign))
		}
	}
	return nil
}

func (t TrackStakeChangesDecorator) checkTotalStakeChange(ctx sdk.Context, totalBondedDelta math.Int) error {
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
	for delegatorKey, delta := range stakeChanges.delegatorBondedDelta {
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
