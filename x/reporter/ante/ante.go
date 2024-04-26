package ante

import (
	"cosmossdk.io/math"
	"github.com/cockroachdb/errors"
	"github.com/tellor-io/layer/x/reporter/keeper"

	"github.com/tellor-io/layer/x/reporter/types"

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
		case *stakingtypes.MsgDelegate:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgBeginRedelegate:
			msgAmount = msg.Amount.Amount
		case *stakingtypes.MsgCancelUnbondingDelegation:
			msgAmount = msg.Amount.Amount
		default:
			continue
		}
		state, err := t.reporterKeeper.Tracker.Get(ctx)
		if err != nil {
			return ctx, err
		}
		currentAmount, err := t.stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return ctx, err
		}
		incr := currentAmount.Add(msgAmount)
		if incr.GT(state.FivePercent) {
			return ctx, errors.New("amount is over the twelve hour five percent limit")
		}
	}

	return next(ctx, tx, simulate)
}
