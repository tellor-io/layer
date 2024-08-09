package ante

import (
	"errors"

	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

func (t TrackStakeChangesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// loop through all the messages and check if the message type will change stake by more than 5%
	var msgAmount math.Int
	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		// case *stakingtypes.MsgCreateValidator:
		// 	msgAmount = msg.Value.Amount
		case *stakingtypes.MsgDelegate:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgBeginRedelegate:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgCancelUnbondingDelegation:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgUndelegate:
			msgAmount = msg.Amount.Amount.Neg()
		default:
			continue
		}
		state, err := t.reporterKeeper.Tracker.Get(ctx)
		if err != nil {
			// for when chain is first started
			if errors.Is(err, collections.ErrNotFound) {
				return ctx, nil
			}
			return ctx, err
		}
		currentAmount, err := t.stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return ctx, err
		}
		if msgAmount.IsNegative() {
			removedFivePercent := state.Amount.Sub(state.Amount.QuoRaw(20))
			lowerBound := currentAmount.Add(msgAmount)
			if lowerBound.LT(removedFivePercent) {
				return ctx, errors.New("amount decreases total stake by more than the allowed 5% in a twelve hour period")
			}
		} else {
			// add 5 percent
			addedFivePercent := state.Amount.Add(state.Amount.QuoRaw(20))
			upperBound := currentAmount.Add(msgAmount)
			if upperBound.GT(addedFivePercent) {
				return ctx, errors.New("amount increases total stake by more than the allowed 5% in a twelve hour period")
			}
		}

	}

	return next(ctx, tx, simulate)
}
