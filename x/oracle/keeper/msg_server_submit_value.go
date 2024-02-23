package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
	regtypes "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	reporterAddr, err := msg.GetSignerAndValidateMsg()
	if err != nil {
		return nil, err
	}
	// get reporter
	reporter, err := k.reporterKeeper.Reporter(ctx, reporterAddr)
	if err != nil {
		return nil, err
	}

	votingPower := reporter.TotalTokens.Quo(sdk.DefaultPowerReduction).Int64()
	// check if querydata has prefix 0x
	msg.QueryData = regtypes.Remove0xPrefix(msg.QueryData)
	// decode query data hex string to bytes
	qDataBytes, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data string: %v", err))
	}
	// get commit from store
	commitValue, err := k.Commits.Get(ctx, collections.Join(reporterAddr.Bytes(), HashQueryData(qDataBytes)))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "no commits to reveal found")
		}
		return nil, err
	}
	currentBlock := ctx.BlockHeight()
	// check if value is being revealed in the one block after commit
	if currentBlock == commitValue.Block {
		return nil, types.ErrCommitRevealWindowEarly
	}
	if currentBlock-1 != commitValue.Block {
		return nil, types.ErrMissedCommitRevealWindow
	}
	// if commitValue.Block < ctx.BlockHeight()-5 || commitValue.Block > ctx.BlockHeight() {
	// 	return nil, status.Error(codes.InvalidArgument, "missed block height window to reveal")
	// }

	commit := utils.CalculateCommitment(msg.Value, msg.Salt)
	if commit != commitValue.Report.Hash {
		return nil, errors.New("submitted value doesn't match commitment, are you a cheater?")
	}

	// set value
	if err := k.setValue(ctx, reporterAddr, msg.Value, qDataBytes, votingPower, currentBlock); err != nil {
		return nil, err
	}
	// emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"NewReport",
			sdk.NewAttribute("reporter", msg.Creator),
			sdk.NewAttribute("query_data", msg.QueryData),
			sdk.NewAttribute("value", msg.Value),
		),
	})

	return &types.MsgSubmitValueResponse{}, nil
}
