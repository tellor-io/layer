package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	msgSender, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid sender address: %v", err))
	}
	valAddr := sdk.ValAddress(msgSender)
	// check if validator is bonded and active
	votingPower, isBonded := k.IsReporterStaked(ctx, valAddr)
	if !isBonded {
		return nil, status.Error(codes.Unauthenticated, "validator is not staked")
	}
	// check if querydata has prefix 0x
	if registryKeeper.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	// decode query data hex string to bytes
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data string: %v", err))
	}
	// get commit from store
	commitValue, err := k.GetSignature(ctx, msg.Creator, HashQueryData(queryData))
	if err != nil {
		return nil, err
	}
	// check if value is being revealed in the one block after commit
	if ctx.BlockHeight()-1 != commitValue.Block {
		return nil, status.Error(codes.InvalidArgument, "missed block height to reveal")
	}
	// if commitValue.Block < ctx.BlockHeight()-5 || commitValue.Block > ctx.BlockHeight() {
	// 	return nil, status.Error(codes.InvalidArgument, "missed block height window to reveal")
	// }
	// verify value signature
	if !k.VerifySignature(ctx, msg.Creator, msg.Value, commitValue.Report.Signature) {
		return nil, status.Error(codes.InvalidArgument, "unable to verify signature")
	}
	// set value
	if err := k.setValue(ctx, msg.Creator, msg.Value, queryData, votingPower); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to set value: %v", err))
	}
	// emit event
	err = ctx.EventManager().EmitTypedEvent(msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitValueResponse{}, nil
}
