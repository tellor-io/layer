package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	reporter := sdk.MustAccAddressFromBech32(msg.Creator)
	// check if validator is bonded and active
	votingPower, isBonded := k.IsReporterStaked(ctx, sdk.ValAddress(reporter))
	if !isBonded {
		return nil, types.ErrValidatorNotBonded
	}
	// check if querydata has prefix 0x
	if registryKeeper.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	// decode query data hex string to bytes
	qDataBytes, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data string: %v", err))
	}
	// get commit from store
	commitValue, err := k.GetSignature(ctx, reporter, HashQueryData(qDataBytes))
	if err != nil {
		return nil, err
	}
	currentBlock := ctx.BlockHeight()
	// check if value is being revealed in the one block after commit
	if currentBlock-1 != commitValue.Block {
		return nil, types.ErrMissedCommitRevealWindow
	}
	// if commitValue.Block < ctx.BlockHeight()-5 || commitValue.Block > ctx.BlockHeight() {
	// 	return nil, status.Error(codes.InvalidArgument, "missed block height window to reveal")
	// }
	// verify value signature
	// if !k.VerifySignature(ctx, msg.Creator, msg.Value, commitValue.Report.Signature) {
	// 	return nil, types.ErrSignatureVerificationFailed
	// }

	// calculate the move's commitment, must match the one stored
	commit := utils.CalculateCommitment(msg.Value, msg.Salt)
	if commit != commitValue.Report.SaltedValue {
		return nil, errors.New("move doesn't match commitment, are you a cheater?")
	}

	// set value
	if err := k.setValue(ctx, reporter, msg.Value, qDataBytes, votingPower, currentBlock); err != nil {
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
