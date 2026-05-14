package ante

import (
	"errors"
	"fmt"
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

type stakeChangeTracker struct {
	totalBondedDelta     math.Int
	delegatorBondedDelta map[string]math.Int
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
		delegatorBondedDelta: make(map[string]math.Int),
	}
}

func (t *stakeChangeTracker) delegatorDelta(delegator sdk.AccAddress) math.Int {
	delta, ok := t.delegatorBondedDelta[string(delegator)]
	if !ok {
		return math.ZeroInt()
	}
	return delta
}

func (t *stakeChangeTracker) add(delegator sdk.AccAddress, amount math.Int) {
	if amount.IsZero() {
		return
	}
	t.totalBondedDelta = t.totalBondedDelta.Add(amount)
	if delegator == nil {
		return
	}
	t.delegatorBondedDelta[string(delegator)] = t.delegatorDelta(delegator).Add(amount)
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
	var msgAmount math.Int
	var delegatorAddr sdk.AccAddress
	switch msg := msg.(type) {
	case *stakingtypes.MsgCreateValidator:
		msgAmount = msg.Value.Amount
		if valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err == nil {
			delegatorAddr = sdk.AccAddress(valAddr)
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
		} else {
			return nil
		}
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
		} else if sourceVal.Status == destVal.Status {
			return nil
		} else if sourceVal.Status != stakingtypes.Bonded && destVal.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
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
		} else {
			return nil
		}
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
		} else {
			return nil
		}
	default:
		return nil
	}

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
			if msgAmount.IsPositive() {
				if err := t.checkDelegatorStakeShare(ctx, delegatorAddr, msgAmount, currentAmount, stakeChanges); err != nil {
					return err
				}
			}
			if stakeChanges != nil {
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
	if msgAmount.IsPositive() {
		if err := t.checkDelegatorStakeShare(ctx, delegatorAddr, msgAmount, currentAmount, stakeChanges); err != nil {
			return err
		}
	}
	if stakeChanges != nil {
		stakeChanges.add(delegatorAddr, msgAmount)
	}
	return nil
}

func (t TrackStakeChangesDecorator) checkDelegatorStakeShare(ctx sdk.Context, delegator sdk.AccAddress, msgAmount math.Int, currentTotalBonded math.Int, stakeChanges *stakeChangeTracker) error {
	if delegator == nil || !currentTotalBonded.IsPositive() {
		return nil
	}
	totalBondedAfter := currentTotalBonded.Add(msgAmount)
	if stakeChanges != nil {
		totalBondedAfter = currentTotalBonded.Add(stakeChanges.totalBondedDelta).Add(msgAmount)
	}
	if !totalBondedAfter.IsPositive() {
		return nil
	}
	currentDelegatorBonded, err := t.delegatorBondedTokens(ctx, delegator)
	if err != nil {
		return err
	}
	delegatorBondedAfter := currentDelegatorBonded.Add(msgAmount)
	if stakeChanges != nil {
		delegatorBondedAfter = currentDelegatorBonded.Add(stakeChanges.delegatorDelta(delegator)).Add(msgAmount)
	}
	if delegatorBondedAfter.MulRaw(10).GT(totalBondedAfter.MulRaw(3)) {
		return types.ErrExceedsMaxStakeShare
	}
	return nil
}

func (t TrackStakeChangesDecorator) delegatorBondedTokens(ctx sdk.Context, delegator sdk.AccAddress) (math.Int, error) {
	tokens := math.ZeroInt()
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
			tokens = tokens.Add(val.TokensFromShares(delegation.Shares).TruncateInt())
		}
		return false
	})
	if err != nil {
		return math.Int{}, err
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
