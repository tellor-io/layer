package ante

import (
	"errors"
	"fmt"

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

func NewTrackStakeChangesDecorator(rk keeper.Keeper, sk types.StakingKeeper) TrackStakeChangesDecorator {
	return TrackStakeChangesDecorator{
		reporterKeeper: rk,
		stakingKeeper:  sk,
	}
}

// implement the AnteDecorator interface
func (t TrackStakeChangesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// check if the message type will change stake by more than 5%
	for _, msg := range tx.GetMsgs() {
		if err := t.processMessage(ctx, msg, 1); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

func (t TrackStakeChangesDecorator) processMessage(ctx sdk.Context, msg sdk.Msg, nestedMsgCount int64) error {
	fmt.Println("nestedMsgCount: ", nestedMsgCount)
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
			if err := t.processMessage(ctx, innerMsg, nestedMsgCount); err != nil {
				return err
			}
		}
	// if the message is not an authz exec, check if it is a stake change message
	default:
		if err := t.checkStakeChange(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (t TrackStakeChangesDecorator) checkStakeChange(ctx sdk.Context, msg sdk.Msg) error {
	var msgAmount math.Int
	switch msg := msg.(type) {
	case *stakingtypes.MsgCreateValidator:
		msgAmount = msg.Value.Amount
	case *stakingtypes.MsgDelegate:
		val, err := t.stakingKeeper.GetValidator(ctx, sdk.ValAddress(msg.ValidatorAddress))
		if err != nil {
			return err
		}
		if val.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
		} else {
			return nil
		}
	case *stakingtypes.MsgBeginRedelegate:
		// redelegate shouldn't increase the total stake, however if its coming from
		// a validator that is not in the active set, it might be considered as an increase
		// in the active stake. Hence, we need to handle it appropriately.
		sourceAddr := msg.ValidatorSrcAddress
		destAddr := msg.ValidatorDstAddress
		sourceVal, err := t.stakingKeeper.GetValidator(ctx, sdk.ValAddress(sourceAddr))
		if err != nil {
			return err
		}
		destVal, err := t.stakingKeeper.GetValidator(ctx, sdk.ValAddress(destAddr))
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
		val, err := t.stakingKeeper.GetValidator(ctx, sdk.ValAddress(msg.ValidatorAddress))
		if err != nil {
			return err
		}
		if val.Status == stakingtypes.Bonded {
			msgAmount = msg.Amount.Amount
		} else {
			return nil
		}
	case *stakingtypes.MsgUndelegate:
		val, err := t.stakingKeeper.GetValidator(ctx, sdk.ValAddress(msg.ValidatorAddress))
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

	// get the total bonded tokens that was set in the last update
	// to compare against the current amount of bonded tokens
	lastupdated, err := t.reporterKeeper.Tracker.Get(ctx)
	if err != nil {
		// for when chain is first started
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}
	currentAmount, err := t.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	changeAmt := currentAmount.Add(msgAmount)
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
	return nil
}
