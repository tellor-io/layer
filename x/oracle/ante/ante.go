package ante

import (
	"fmt"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

const (
	MaxNestedMsgCount = 7
)

// TrackStakeChangesDecorator is an AnteDecorator that checks if the transaction is going to change stake by more than 5% and disallows the transaction to enter the mempool or be executed if so
type TrackNoStakeReportingDecorator struct {
	oracleKeeper keeper.Keeper
}

func NewTrackNoStakeReportingDecorator(ok keeper.Keeper) TrackNoStakeReportingDecorator {
	return TrackNoStakeReportingDecorator{
		oracleKeeper: ok,
	}
}

// implement the AnteDecorator interface
func (t TrackNoStakeReportingDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// check that no more than one no stake report per queryId is being made
	for _, msg := range tx.GetMsgs() {
		if err := t.processMessage(ctx, msg, 1); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

func (t TrackNoStakeReportingDecorator) processMessage(ctx sdk.Context, msg sdk.Msg, nestedMsgCount int64) error {
	if nestedMsgCount > MaxNestedMsgCount {
		return fmt.Errorf("nested message count exceeds the maximum allowed: Limit is %d", MaxNestedMsgCount)
	}
	switch msg := msg.(type) {
	// if the message is an authz exec, check the inner messages for any no stake reports
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
	// if the message is not an authz exec, check if it is a no stake report message
	default:
		if err := t.checkNoStakeReport(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (t TrackNoStakeReportingDecorator) checkNoStakeReport(ctx sdk.Context, msg sdk.Msg) error {
	switch msg := msg.(type) {
	case *types.MsgNoStakeReport:
		queryData := msg.QueryData
		queryId := utils.QueryIDFromData(queryData)
		currentHeight := uint64(ctx.BlockHeight())
		key := collections.Join(currentHeight, queryId)
		_, err := t.oracleKeeper.NoStakeTracker.Get(ctx, key)
		if err == nil {
			return fmt.Errorf("no stake report already exists for this queryId at height: %d, please submit again next block", currentHeight)
		}
		return nil
	default:
		return nil
	}
}
