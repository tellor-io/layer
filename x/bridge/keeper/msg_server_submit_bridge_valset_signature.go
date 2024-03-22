package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitBridgeValsetSignature(ctx context.Context, msg *types.MsgSubmitBridgeValsetSignature) (*types.MsgSubmitBridgeValsetSignatureResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	operatorAddr, err := convertPrefix(msg.Creator, "tellorvaloper")
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to convert operator address prefix", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	timestampUint64, err := strconv.ParseUint(msg.Timestamp, 10, 64)
	if err != nil {
		k.Keeper.Logger(sdkCtx).Error("failed to parse timestamp", "error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	k.Keeper.SetBridgeValsetSignature(sdkCtx, operatorAddr, timestampUint64, msg.Signature)

	return &types.MsgSubmitBridgeValsetSignatureResponse{}, nil
}
