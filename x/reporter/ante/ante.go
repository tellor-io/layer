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

// implement the AnteDecorator interface
func (t TrackStakeChangesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// loop through all the messages and check if the message type will change stake by more than 5%
	var msgAmount math.Int
	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgCreateValidator:
			msgAmount = msg.Value.Amount
		case *stakingtypes.MsgDelegate:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgBeginRedelegate:
			// redelegate shouldn't increase the total stake, however if its coming from
			// a validator that is not in the active set, it might be considered as an increase
			// in the active stake. Hence, we need to handle it appropriately.
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgCancelUnbondingDelegation:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgUndelegate:
			// negate the amount since undelegating is removing stake from the chain
			// and to help with the comparison later on
			msgAmount = msg.Amount.Amount.Neg()
		default:
			continue
		}
		// get the total bonded tokens that was set in the last update
		// to compare against the current amount of bonded tokens
		lastupdated, err := t.reporterKeeper.Tracker.Get(ctx)
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
		changeAmt := currentAmount.Add(msgAmount)
		if msgAmount.IsNegative() {
			// subtract 5 percent from last updated amount
			allowedLowerBound := lastupdated.Amount.Sub(lastupdated.Amount.QuoRaw(20))
			if changeAmt.LT(allowedLowerBound) {
				return ctx, errors.New("total stake decrease exceeds the allowed 5% threshold within a twelve-hour period")
			}
		} else {
			// add 5 percent to last updated amount
			allowedUpperBound := lastupdated.Amount.Add(lastupdated.Amount.QuoRaw(20))
			if changeAmt.GT(allowedUpperBound) {
				return ctx, errors.New("total stake increase exceeds the allowed 5% threshold within a twelve-hour period")
			}
		}

	}

	return next(ctx, tx, simulate)
}
